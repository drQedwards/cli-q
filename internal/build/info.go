// Package build exposes version information injected at build time via ldflags.
package build

// Injected by GoReleaser (or make build) via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
