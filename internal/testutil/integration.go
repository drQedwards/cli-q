// Package testutil provides shared helpers for integration tests.
package testutil

import (
	"archive/zip"
	"bytes"
	"os"
	"testing"

	"github.com/supermodeltools/cli/internal/config"
)

// IntegrationConfig returns a Config for integration tests and skips the test
// if no API key is available. Tests that call this are safe to run in CI by
// setting SUPERMODEL_API_KEY; on developer machines the local config file is
// used as a fallback.
func IntegrationConfig(t *testing.T) *config.Config {
	t.Helper()

	if key := os.Getenv("SUPERMODEL_API_KEY"); key != "" {
		base := os.Getenv("SUPERMODEL_API_BASE")
		if base == "" {
			base = config.DefaultAPIBase
		}
		return &config.Config{APIKey: key, APIBase: base}
	}

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("skipping integration test: could not load config: %v", err)
	}
	if cfg.APIKey == "" {
		t.Skip("skipping integration test: no API key (set SUPERMODEL_API_KEY or run `supermodel login`)")
	}
	return cfg
}

// MinimalGoZip writes a tiny Go repository into a temp ZIP file and returns
// its path. The caller is responsible for removing it.
func MinimalGoZip(t *testing.T) string {
	t.Helper()

	f, err := os.CreateTemp("", "supermodel-integration-*.zip")
	if err != nil {
		t.Fatalf("create temp zip: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	files := map[string]string{
		"go.mod": "module example.com/hello\n\ngo 1.21\n",
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println(greet("world"))
}

func greet(name string) string {
	return "Hello, " + name + "!"
}
`,
		"greet_test.go": `package main

import "testing"

func TestGreet(t *testing.T) {
	if got := greet("test"); got != "Hello, test!" {
		t.Fatalf("got %q", got)
	}
}
`,
	}

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	// Verify the ZIP is valid.
	stat, _ := f.Stat()
	data, _ := os.ReadFile(f.Name())
	if _, err := zip.NewReader(bytes.NewReader(data), stat.Size()); err != nil {
		t.Fatalf("produced invalid zip: %v", err)
	}

	return f.Name()
}

// MinimalGoDir creates a temp directory with a tiny Go repository and returns
// its path. Useful when commands need a directory rather than a ZIP.
func MinimalGoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"go.mod": "module example.com/hello\n\ngo 1.21\n",
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println(greet("world"))
}

func greet(name string) string {
	return "Hello, " + name + "!"
}
`,
	}
	for name, content := range files {
		if err := os.WriteFile(dir+"/"+name, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}
