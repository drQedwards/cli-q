package audit

import "github.com/supermodeltools/cli/internal/gitzip"

// CreateZip archives the repository at dir into a temporary ZIP file and
// returns its path. The caller is responsible for removing the file.
func CreateZip(dir string) (string, error) {
	return gitzip.CreateZip(dir, "supermodel-factory-*.zip")
}
