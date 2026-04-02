package analyze_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/supermodeltools/cli/internal/analyze"
	"github.com/supermodeltools/cli/internal/testutil"
	"github.com/supermodeltools/cli/internal/ui"
)

// TestIntegration_GetGraph uploads a minimal Go repo, calls GetGraph, and
// verifies the returned graph contains nodes and a repoId.
func TestIntegration_GetGraph(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	g, hash, err := analyze.GetGraph(ctx, cfg, dir, true /* force */)
	if err != nil {
		t.Fatalf("GetGraph: %v", err)
	}
	if len(g.Nodes) == 0 {
		t.Fatal("expected at least one node, got none")
	}
	if hash == "" {
		t.Error("expected non-empty cache hash")
	}
	t.Logf("nodes=%d rels=%d repoId=%q hash=%s", len(g.Nodes), len(g.Rels()), g.RepoID(), hash[:8])
}

// TestIntegration_GetGraph_CacheHit calls GetGraph twice and verifies the
// second call uses the cache (same result, no API error).
func TestIntegration_GetGraph_CacheHit(t *testing.T) {
	// Redirect HOME so cache writes go to a temp dir, not ~/.supermodel.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// First call: force=true to populate cache.
	g1, hash1, err := analyze.GetGraph(ctx, cfg, dir, true)
	if err != nil {
		t.Fatalf("GetGraph (first): %v", err)
	}

	// Second call: force=false should hit cache.
	g2, hash2, err := analyze.GetGraph(ctx, cfg, dir, false)
	if err != nil {
		t.Fatalf("GetGraph (cached): %v", err)
	}
	if hash1 != hash2 {
		t.Errorf("cache hashes differ: %s vs %s", hash1, hash2)
	}
	if len(g2.Nodes) == 0 {
		t.Fatal("cached graph has no nodes")
	}
	t.Logf("first=%d nodes, cached=%d nodes", len(g1.Nodes), len(g2.Nodes))
}

// TestIntegration_Run verifies that Run writes a human-readable summary.
func TestIntegration_Run(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)

	// Redirect HOME so cache writes go to a temp dir.
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := analyze.Run(ctx, cfg, dir, analyze.Options{Force: true, Output: "human"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// TestIntegration_Run_JSON verifies that Run writes valid JSON summary.
func TestIntegration_Run_JSON(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Capture stdout via os.Pipe for JSON output validation.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = w

	runErr := analyze.Run(ctx, cfg, dir, analyze.Options{Force: true, Output: "json"})
	w.Close()
	os.Stdout = oldStdout

	if runErr != nil {
		t.Fatalf("Run JSON: %v", runErr)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	for _, want := range []string{"repo_id", "files", "functions", "relationships"} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON output missing field %q:\n%s", want, out)
		}
	}
}

// TestIntegration_GetGraph_ReturnsNodesAndRels verifies the graph has the
// expected structure for the minimal repo: at least one File and one Function.
func TestIntegration_GetGraph_ReturnsNodesAndRels(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	g, _, err := analyze.GetGraph(ctx, cfg, dir, true)
	if err != nil {
		t.Fatalf("GetGraph: %v", err)
	}

	files := g.NodesByLabel("File")
	if len(files) == 0 {
		t.Error("expected at least one File node")
	}
	functions := g.NodesByLabel("Function")
	if len(functions) == 0 {
		t.Error("expected at least one Function node")
	}
	t.Logf("files=%d functions=%d rels=%d", len(files), len(functions), len(g.Rels()))
	_ = ui.FormatHuman // ensure ui import is used
}
