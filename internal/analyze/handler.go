package analyze

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/supermodeltools/cli/internal/api"
	"github.com/supermodeltools/cli/internal/build"
	"github.com/supermodeltools/cli/internal/cache"
	"github.com/supermodeltools/cli/internal/config"
	"github.com/supermodeltools/cli/internal/ui"
)

// Options configures the analyze command.
type Options struct {
	Force  bool   // bypass cache and re-upload
	Output string // "human" | "json"
}

// Run archives dir, uploads it to the Supermodel API, caches the result, and
// prints a summary. Uses the cache when available unless Force is set.
func Run(ctx context.Context, cfg *config.Config, dir string, opts Options) error {
	g, _, err := GetGraph(ctx, cfg, dir, opts.Force)
	if err != nil {
		return err
	}
	return printSummary(os.Stdout, g, ui.ParseFormat(opts.Output))
}

// GetGraph returns the display graph for dir, running analysis if the cache
// is cold or force is true. It returns the graph and the cache key.
//
// Uses git-based fingerprinting (~1ms for clean repos) to check the cache
// before creating a zip. Only creates and uploads the zip on cache miss.
func GetGraph(ctx context.Context, cfg *config.Config, dir string, force bool) (*api.Graph, string, error) { //nolint:gocyclo // orchestration: multiple cache checks, fallback paths, and error branches are inherently complex
	if err := guardDir(dir); err != nil {
		return nil, "", err
	}

	// Fast path: check cache using git fingerprint before creating zip.
	if !force {
		fingerprint, err := cache.RepoFingerprint(dir)
		if err == nil {
			key := cache.AnalysisKey(fingerprint, "graph", build.Version)
			if g, _ := cache.Get(key); g != nil {
				ui.Success("Using cached analysis (repoId: %s)", g.RepoID())
				return g, key, nil
			}
		}
	}

	// Cache miss: create zip and upload.
	spin := ui.Start("Creating repository archive…")
	zipPath, err := createZip(dir)
	spin.Stop()
	if err != nil {
		return nil, "", fmt.Errorf("create archive: %w", err)
	}
	defer os.Remove(zipPath)

	if info, statErr := os.Stat(zipPath); statErr == nil {
		sizeMB := float64(info.Size()) / (1 << 20)
		if sizeMB > 50 {
			ui.Warn("Archive is %.0f MB — add patterns to .supermodel.json to reduce upload size", sizeMB)
		}
	}

	hash, err := cache.HashFile(zipPath)
	if err != nil {
		return nil, "", err
	}

	// Also check by zip hash (covers non-git repos and fingerprint edge cases).
	if !force {
		if g, _ := cache.Get(hash); g != nil {
			ui.Success("Using cached analysis (repoId: %s)", g.RepoID())
			return g, hash, nil
		}
	}

	client := api.New(cfg)
	spin = ui.Start("Uploading and analyzing repository…")
	ir, err := client.AnalyzeShards(ctx, zipPath, "analyze-"+hash[:16], nil)
	spin.Stop()
	if err != nil {
		// Network/API failure — serve stale cache if available rather than hard-failing.
		if stale, _ := cache.Get(hash); stale != nil {
			ui.Warn("API unavailable (%v) — using stale cached result", err)
			return stale, hash, nil
		}
		if fp, fpErr := cache.RepoFingerprint(dir); fpErr == nil {
			fpKey := cache.AnalysisKey(fp, "graph", build.Version)
			if stale, _ := cache.Get(fpKey); stale != nil {
				ui.Warn("API unavailable (%v) — using stale cached result", err)
				return stale, fpKey, nil
			}
		}
		return nil, hash, err
	}

	g := api.GraphFromShardIR(ir)

	// Cache under both keys: fingerprint (fast lookup) and zip hash (fallback).
	fingerprint, fpErr := cache.RepoFingerprint(dir)
	if fpErr == nil {
		fpKey := cache.AnalysisKey(fingerprint, "graph", build.Version)
		if err := cache.Put(fpKey, g); err != nil {
			ui.Warn("could not write cache: %v", err)
		}
	}
	if err := cache.Put(hash, g); err != nil {
		ui.Warn("could not write cache: %v", err)
	}

	// Also populate the shard cache (.supermodel/graph.json) so that
	// files.Generate() called after analyze reuses this result without a
	// second API upload.
	absDir, _ := filepath.Abs(dir)
	shardCacheFile := filepath.Join(absDir, ".supermodel", "shards.json")
	if irJSON, marshalErr := json.MarshalIndent(ir, "", "  "); marshalErr == nil {
		if mkErr := os.MkdirAll(filepath.Dir(shardCacheFile), 0o755); mkErr == nil {
			tmp := shardCacheFile + ".tmp"
			if writeErr := os.WriteFile(tmp, irJSON, 0o644); writeErr == nil {
				_ = os.Rename(tmp, shardCacheFile)
			}
		}
	}

	ui.Success("Analysis complete (repoId: %s)", g.RepoID())
	return g, hash, nil
}

type summary struct {
	RepoID        string `json:"repo_id"`
	Files         int    `json:"files"`
	Functions     int    `json:"functions"`
	Relationships int    `json:"relationships"`
}

func printSummary(w io.Writer, g *api.Graph, fmt_ ui.Format) error {
	s := summary{
		RepoID:        g.RepoID(),
		Files:         len(g.NodesByLabel("File")),
		Functions:     len(g.NodesByLabel("Function")),
		Relationships: len(g.Rels()),
	}
	if fmt_ == ui.FormatJSON {
		return ui.JSON(w, s)
	}
	ui.Table(w, []string{"FIELD", "VALUE"}, [][]string{
		{"Repo ID", s.RepoID},
		{"Files", fmt.Sprintf("%d", s.Files)},
		{"Functions", fmt.Sprintf("%d", s.Functions)},
		{"Relationships", fmt.Sprintf("%d", s.Relationships)},
	})
	return nil
}
