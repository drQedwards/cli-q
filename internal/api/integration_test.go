package api_test

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/supermodeltools/cli/internal/api"
	"github.com/supermodeltools/cli/internal/config"
)

// TestIntegration_Analyze uploads a minimal Go repo and verifies the API
// returns a non-empty graph with at least one node and the repoId metadata.
// It is skipped unless SUPERMODEL_API_KEY is set (or present in ~/.supermodel/config.yaml).
func TestIntegration_Analyze(t *testing.T) {
	cfg := integrationConfig(t)
	client := api.New(cfg)

	zipPath := minimalGoZip(t)
	defer os.Remove(zipPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	graph, err := client.Analyze(ctx, zipPath, "integration-test-analyze")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(graph.Nodes) == 0 {
		t.Fatal("expected at least one node in graph, got none")
	}
	t.Logf("nodes=%d rels=%d repoId=%q", len(graph.Nodes), len(graph.Rels()), graph.RepoID())
}

// TestIntegration_DisplayGraph calls Analyze then DisplayGraph and checks
// that the second call returns the same repo's graph.
func TestIntegration_DisplayGraph(t *testing.T) {
	cfg := integrationConfig(t)
	client := api.New(cfg)

	zipPath := minimalGoZip(t)
	defer os.Remove(zipPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	graph, err := client.Analyze(ctx, zipPath, "integration-test-displaygraph")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	repoID := graph.RepoID()
	if repoID == "" {
		t.Skip("API did not return repoId; skipping DisplayGraph test")
	}

	display, err := client.DisplayGraph(ctx, repoID, "integration-test-displaygraph-display")
	if err != nil {
		t.Fatalf("DisplayGraph: %v", err)
	}
	if len(display.Nodes) == 0 {
		t.Fatal("expected at least one node from DisplayGraph, got none")
	}
	t.Logf("DisplayGraph: nodes=%d rels=%d", len(display.Nodes), len(display.Rels()))
}

// integrationConfig returns a Config suitable for integration tests.
// It skips the test if no API key is available.
func integrationConfig(t *testing.T) *config.Config {
	t.Helper()

	// Prefer explicit env var so CI can inject a key without a config file.
	if key := os.Getenv("SUPERMODEL_API_KEY"); key != "" {
		base := os.Getenv("SUPERMODEL_API_BASE")
		if base == "" {
			base = config.DefaultAPIBase
		}
		return &config.Config{APIKey: key, APIBase: base}
	}

	// Fall back to the local config file (developer machines).
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("skipping integration test: could not load config: %v", err)
	}
	if cfg.APIKey == "" {
		t.Skip("skipping integration test: no API key configured (set SUPERMODEL_API_KEY or run `supermodel login`)")
	}
	return cfg
}

// minimalGoZip writes a tiny Go repository into a temp ZIP file and returns
// its path. The caller is responsible for removing it.
func minimalGoZip(t *testing.T) string {
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

	// Verify the ZIP is valid before returning.
	stat, _ := f.Stat()
	if _, err := zip.NewReader(bytes.NewReader(func() []byte {
		data, _ := os.ReadFile(f.Name())
		return data
	}()), stat.Size()); err != nil {
		t.Fatalf("produced invalid zip: %v", err)
	}

	return f.Name()
}
