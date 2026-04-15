// Package gitzip creates ZIP archives of source trees while respecting
// .gitignore. Three strategies are tried in order:
//
//  1. git archive HEAD  — clean git repo; fastest, fully deterministic.
//  2. git ls-files      — dirty git repo; respects .gitignore on all platforms.
//  3. filepath.Walk     — non-git directory; uses a built-in skip list.
package gitzip

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SkipDirs are directory names excluded from the fallback walk (strategy 3).
// Strategies 1 and 2 rely on git for exclusion and ignore this list.
var skipDirs = map[string]bool{
	".git":             true,
	".claude":          true,
	".idea":            true,
	".vscode":          true,
	".cache":           true,
	".turbo":           true,
	".nx":              true,
	".next":            true,
	".nuxt":            true,
	".terraform":       true,
	".tox":             true,
	".venv":            true,
	".pnpm-store":      true,
	"__pycache__":      true,
	"__snapshots__":    true,
	"bower_components": true,
	"build":            true,
	"coverage":         true,
	"dist":             true,
	"node_modules":     true,
	"out":              true,
	"target":           true,
	"vendor":           true,
	"venv":             true,
}

// CreateZip archives the source tree at dir into a temporary ZIP file and
// returns its path. pattern is passed to os.CreateTemp (e.g. "supermodel-*.zip").
// The caller is responsible for removing the file.
func CreateZip(dir, pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	dest := f.Name()
	f.Close()

	if isGitRepo(dir) {
		if isWorktreeClean(dir) {
			if err := gitArchive(dir, dest); err == nil {
				return dest, nil
			}
		}
		// Dirty worktree: use git ls-files so .gitignore is still respected.
		if err := gitLsFilesZip(dir, dest); err == nil {
			return dest, nil
		}
	}

	if err := walkZip(dir, dest); err != nil {
		os.Remove(dest)
		return "", err
	}
	return dest, nil
}

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir") //nolint:gosec // dir is user-supplied cwd
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func isWorktreeClean(dir string) bool {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output() //nolint:gosec // dir is user-supplied cwd
	return err == nil && strings.TrimSpace(string(out)) == ""
}

func gitArchive(dir, dest string) error {
	cmd := exec.Command("git", "-C", dir, "archive", "--format=zip", "-o", dest, "HEAD") //nolint:gosec // dir is user-supplied cwd; dest is temp file
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitLsFilesZip builds a ZIP from the output of `git ls-files -co
// --exclude-standard`, which lists tracked files plus untracked files that are
// not gitignored. This preserves .gitignore semantics for dirty worktrees.
func gitLsFilesZip(dir, dest string) error {
	out, err := exec.Command("git", "-C", dir, "ls-files", "-co", "--exclude-standard").Output() //nolint:gosec // dir is user-supplied cwd
	if err != nil {
		return err
	}

	f, err := os.Create(dest) //nolint:gosec // dest is a temp file path from os.CreateTemp
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		rel := scanner.Text()
		if rel == "" {
			continue
		}
		absPath := filepath.Join(dir, filepath.FromSlash(rel))
		info, err := os.Lstat(absPath)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Size() > 10<<20 {
			continue
		}
		w, err := zw.Create(rel) // always use slash separators in ZIP
		if err != nil {
			return err
		}
		if err := copyFile(absPath, w); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// walkZip creates a ZIP of dir using a directory walk. Used only when git is
// not available. Excludes skipDirs, hidden files, symlinks, and files > 10 MB.
func walkZip(dir, dest string) error {
	out, err := os.Create(dest) //nolint:gosec // dest is a temp file path from os.CreateTemp
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") || info.Size() > 10<<20 {
			return nil
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		return copyFile(path, w)
	})
}

func copyFile(path string, w io.Writer) error {
	f, err := os.Open(path) //nolint:gosec // path is from git ls-files or filepath.Walk within dir
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	f.Close()
	return err
}
