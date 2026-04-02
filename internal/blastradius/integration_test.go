package blastradius_test

import (
	"context"
	"testing"
	"time"

	"github.com/supermodeltools/cli/internal/analyze"
	"github.com/supermodeltools/cli/internal/blastradius"
	"github.com/supermodeltools/cli/internal/testutil"
)

// TestIntegration_Run_KnownFile analyzes the minimal repo, picks a File node
// from the returned graph, and runs blast-radius against it.
func TestIntegration_Run_KnownFile(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// First, get the graph so we know a real file path.
	g, _, err := analyze.GetGraph(ctx, cfg, dir, true)
	if err != nil {
		t.Fatalf("GetGraph: %v", err)
	}
	files := g.NodesByLabel("File")
	if len(files) == 0 {
		t.Skip("no File nodes in graph — cannot run blast-radius test")
	}

	// Pick any file in the graph.
	target := files[0].Prop("path", "name", "file")
	if target == "" {
		t.Skip("File node has no path property")
	}
	t.Logf("running blast-radius for target: %s", target)

	// Run blast-radius. Even if nothing imports this file, it should succeed
	// (zero results is valid).
	err = blastradius.Run(ctx, cfg, dir, target, blastradius.Options{
		Force:  false, // use the cached graph from GetGraph above
		Output: "human",
	})
	if err != nil {
		t.Fatalf("blastradius.Run: %v", err)
	}
}

// TestIntegration_Run_JSON verifies JSON output mode.
func TestIntegration_Run_JSON(t *testing.T) {
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
		t.Skip("no File nodes in graph")
	}
	target := files[0].Prop("path", "name", "file")
	if target == "" {
		t.Skip("File node has no path property")
	}

	err = blastradius.Run(ctx, cfg, dir, target, blastradius.Options{
		Force:  false,
		Output: "json",
	})
	if err != nil {
		t.Fatalf("blastradius.Run JSON: %v", err)
	}
}

// TestIntegration_Run_UnknownFile verifies that an unknown file returns an
// error with a helpful message.
func TestIntegration_Run_UnknownFile(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := blastradius.Run(ctx, cfg, dir, "nonexistent/file.go", blastradius.Options{
		Force: true,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	t.Logf("got expected error: %v", err)
}
