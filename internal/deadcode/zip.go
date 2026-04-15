package deadcode

import "github.com/supermodeltools/cli/internal/gitzip"

// createZip archives the repository at dir into a temporary ZIP file and
// returns its path. The caller is responsible for removing the file.
func createZip(dir string) (string, error) {
	return gitzip.CreateZip(dir, "supermodel-*.zip")
}
