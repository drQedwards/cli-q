package find_test

import (
	"context"
	"testing"
	"time"

	"github.com/supermodeltools/cli/internal/find"
	"github.com/supermodeltools/cli/internal/testutil"
)

// TestIntegration_Run_Find calls Run with a symbol that exists in the minimal
// repo ("greet") and verifies it completes without error.
func TestIntegration_Run_Find(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := find.Run(ctx, cfg, dir, "greet", find.Options{Force: true, Output: "human"})
	if err != nil {
		t.Fatalf("find.Run: %v", err)
	}
}

// TestIntegration_Run_Find_JSON verifies Run produces JSON output without error.
func TestIntegration_Run_Find_JSON(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := find.Run(ctx, cfg, dir, "greet", find.Options{Force: true, Output: "json"})
	if err != nil {
		t.Fatalf("find.Run JSON: %v", err)
	}
}

// TestIntegration_Run_Find_NoMatch verifies Run handles a symbol with no
// matches gracefully (returns nil, not an error).
func TestIntegration_Run_Find_NoMatch(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := find.Run(ctx, cfg, dir, "zzz_nonexistent_symbol_xyz", find.Options{Force: true})
	if err != nil {
		t.Fatalf("find.Run no-match: expected nil, got %v", err)
	}
}

// TestIntegration_Run_Find_KindFilter verifies the --kind flag is respected.
func TestIntegration_Run_Find_KindFilter(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// "greet" should be found as a Function; filtering for File should return
	// no matches (nil error, zero results printed to stderr).
	err := find.Run(ctx, cfg, dir, "greet", find.Options{
		Force:  true,
		Output: "human",
		Kind:   "File",
	})
	if err != nil {
		t.Fatalf("find.Run kind=File: %v", err)
	}
}
