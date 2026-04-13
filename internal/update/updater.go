// Package update implements self-update for the supermodel binary.
//
// It fetches the latest release from GitHub, downloads the archive for the
// current OS/arch, extracts the binary, and atomically replaces the running
// executable.
package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/supermodeltools/cli/internal/build"
)

const releaseAPI = "https://api.github.com/repos/supermodeltools/cli/releases/latest"

// Release is the subset of the GitHub releases API response we need.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset is a single release artifact.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Check fetches the latest release tag and returns it. Does not download anything.
func Check() (latest string, err error) {
	r, err := fetchRelease()
	if err != nil {
		return "", err
	}
	return r.TagName, nil
}

// Run checks for a newer release and, if found, downloads and installs it.
// Returns (true, nil) when an update was installed, (false, nil) when already
// up to date, and (false, err) on failure.
func Run() (updated bool, err error) {
	rel, err := fetchRelease()
	if err != nil {
		return false, fmt.Errorf("fetch release info: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	current := strings.TrimPrefix(build.Version, "v")

	if current == latest || current == "dev" && latest == "" {
		return false, nil
	}
	if current == latest {
		return false, nil
	}

	assetName := assetFilename()
	var target *Asset
	for i := range rel.Assets {
		if rel.Assets[i].Name == assetName {
			target = &rel.Assets[i]
			break
		}
	}
	if target == nil {
		return false, fmt.Errorf("no release asset found for %s (looked for %q)", runtime.GOOS+"/"+runtime.GOARCH, assetName)
	}

	exe, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("locate current executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return false, fmt.Errorf("resolve symlinks: %w", err)
	}

	tmp, err := downloadBinary(target.BrowserDownloadURL, assetName)
	if err != nil {
		return false, fmt.Errorf("download update: %w", err)
	}
	defer os.Remove(tmp)

	if err := install(tmp, exe); err != nil {
		return false, fmt.Errorf("install update: %w", err)
	}
	return true, nil
}

// fetchRelease retrieves the latest GitHub release metadata.
func fetchRelease() (*Release, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, releaseAPI, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("parse release JSON: %w", err)
	}
	return &rel, nil
}

// assetFilename returns the expected archive name for the current platform,
// matching the goreleaser name_template "supermodel_{{ .Os }}_{{ .Arch }}".
func assetFilename() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goos == "windows" {
		return fmt.Sprintf("supermodel_%s_%s.zip", goos, goarch)
	}
	return fmt.Sprintf("supermodel_%s_%s.tar.gz", goos, goarch)
}

// downloadBinary downloads the release archive, extracts the binary into a
// temp file, and returns its path.
func downloadBinary(url, assetName string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "supermodel-update-*")
	if err != nil {
		return "", err
	}
	tmp.Close()

	if strings.HasSuffix(assetName, ".zip") {
		// Download to a temp archive, then extract
		archiveTmp, err := os.CreateTemp("", "supermodel-archive-*.zip")
		if err != nil {
			return "", err
		}
		archiveName := archiveTmp.Name()
		defer os.Remove(archiveName)
		if _, err := io.Copy(archiveTmp, resp.Body); err != nil {
			archiveTmp.Close()
			return "", err
		}
		archiveTmp.Close()
		if err := extractZip(archiveName, tmp.Name()); err != nil {
			return "", err
		}
	} else {
		if err := extractTarGz(resp.Body, tmp.Name()); err != nil {
			return "", err
		}
	}

	return tmp.Name(), nil
}

func extractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Base(hdr.Name)
		if name != "supermodel" {
			continue
		}
		out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, tr) //nolint:gosec
		out.Close()
		return err
	}
	return fmt.Errorf("supermodel binary not found in archive")
}

func extractZip(archivePath, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		name := filepath.Base(f.Name)
		if name != "supermodel.exe" && name != "supermodel" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc) //nolint:gosec
		rc.Close()
		out.Close()
		return err
	}
	return fmt.Errorf("supermodel binary not found in zip archive")
}

// install atomically replaces the current executable with the new binary.
// On Unix this is: rename(tmp → exe.old) then rename(new → exe).
func install(newBin, exe string) error {
	if err := os.Chmod(newBin, 0o755); err != nil {
		return err
	}
	// Atomic replace: write to a sibling temp path then rename over the original.
	dir := filepath.Dir(exe)
	staged, err := os.CreateTemp(dir, ".supermodel-update-*")
	if err != nil {
		// Fallback: write directly (non-atomic but best-effort)
		return replaceFile(newBin, exe)
	}
	staged.Close()
	stagedName := staged.Name()
	defer os.Remove(stagedName)

	if err := copyFile(newBin, stagedName); err != nil {
		return err
	}
	if err := os.Chmod(stagedName, 0o755); err != nil {
		return err
	}
	return os.Rename(stagedName, exe)
}

func replaceFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
