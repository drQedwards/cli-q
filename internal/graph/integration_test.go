//go:build integration

package graph_test

import (
	"context"
	"testing"
	"time"

	"github.com/supermodeltools/cli/internal/graph"
	"github.com/supermodeltools/cli/internal/testutil"
)

// TestIntegration_Run_Human verifies that Run produces human-readable output
// for the minimal Go repo without error.
func TestIntegration_Run_Human(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := graph.Run(ctx, cfg, dir, graph.Options{Force: true, Output: "human"})
	if err != nil {
		t.Fatalf("graph.Run human: %v", err)
	}
}

// TestIntegration_Run_JSON verifies Run produces JSON output without error.
func TestIntegration_Run_JSON(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := graph.Run(ctx, cfg, dir, graph.Options{Force: true, Output: "json"})
	if err != nil {
		t.Fatalf("graph.Run json: %v", err)
	}
}

// TestIntegration_Run_DOT verifies Run produces DOT format output without error.
func TestIntegration_Run_DOT(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := graph.Run(ctx, cfg, dir, graph.Options{Force: true, Output: "dot"})
	if err != nil {
		t.Fatalf("graph.Run dot: %v", err)
	}
}

// TestIntegration_Run_WithFilter verifies Run respects the Filter option.
func TestIntegration_Run_WithFilter(t *testing.T) {
	cfg := testutil.IntegrationConfig(t)
	dir := testutil.MinimalGoDir(t)
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, filter := range []string{"File", "Function"} {
		t.Run("filter="+filter, func(t *testing.T) {
			err := graph.Run(ctx, cfg, dir, graph.Options{
				Force:  true,
				Output: "human",
				Filter: filter,
			})
			if err != nil {
				t.Fatalf("graph.Run filter=%s: %v", filter, err)
			}
		})
	}
}
