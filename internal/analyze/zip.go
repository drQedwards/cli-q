package analyze

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supermodeltools/cli/internal/gitzip"
)

// guardDir returns an error if dir is the filesystem root or the user's home
// directory — running analysis there would upload far too much.
func guardDir(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	// Reject root (Unix "/" or Windows "C:\")
	vol := filepath.VolumeName(abs)
	if abs == vol+string(filepath.Separator) || abs == string(filepath.Separator) {
		return fmt.Errorf("refusing to run in root directory — specify a project directory")
	}
	home, _ := os.UserHomeDir()
	if home != "" && abs == home {
		return fmt.Errorf("refusing to run in home directory (%s) — specify a project directory with --dir", abs)
	}
	return nil
}

// createZip archives the repository at dir into a temporary ZIP file and
// returns its path. The caller is responsible for removing the file.
func createZip(dir string) (string, error) {
	return gitzip.CreateZip(dir, "supermodel-*.zip")
}
