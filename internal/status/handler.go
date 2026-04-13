package status

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/supermodeltools/cli/internal/build"
	"github.com/supermodeltools/cli/internal/cache"
	"github.com/supermodeltools/cli/internal/config"
	"github.com/supermodeltools/cli/internal/ui"
)

// Options configures the status command.
type Options struct {
	Output string // "human" | "json"
}

// Report holds all status information.
type Report struct {
	Version        string    `json:"version"`
	Authed         bool      `json:"authenticated"`
	APIBase        string    `json:"api_base"`
	ConfigPath     string    `json:"config_path"`
	CacheDir       string    `json:"cache_dir"`
	CacheCount     int       `json:"cached_analyses"`
	CacheSizeBytes int64     `json:"cache_size_bytes"`
	LastAnalysis   time.Time `json:"last_analysis,omitempty"`
}

// Run prints the current Supermodel status.
func Run(_ context.Context, opts Options) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	count, sizeBytes := cache.Stats()
	r := Report{
		Version:        build.Version,
		Authed:         cfg.APIKey != "",
		APIBase:        cfg.APIBase,
		ConfigPath:     config.Path(),
		CacheDir:       config.Dir() + "/cache",
		CacheCount:     count,
		CacheSizeBytes: sizeBytes,
		LastAnalysis:   cache.NewestEntry(),
	}
	return render(os.Stdout, &r, ui.ParseFormat(opts.Output))
}

// countCacheEntries counts JSON files in dir. Kept for tests.
func countCacheEntries(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			n++
		}
	}
	return n
}

func render(w io.Writer, r *Report, fmt_ ui.Format) error {
	if fmt_ == ui.FormatJSON {
		return ui.JSON(w, r)
	}

	authed := "Not authenticated — run `supermodel login`"
	if r.Authed {
		authed = "Authenticated"
	}

	cacheStr := fmt.Sprintf("%s  (%d entries, %s)", r.CacheDir, r.CacheCount, formatBytes(r.CacheSizeBytes))
	lastStr := "never"
	if !r.LastAnalysis.IsZero() {
		lastStr = r.LastAnalysis.Format("2006-01-02 15:04:05")
	}

	ui.Table(w, []string{"KEY", "VALUE"}, [][]string{
		{"Version", r.Version},
		{"Auth", authed},
		{"Config", r.ConfigPath},
		{"API base", r.APIBase},
		{"Cache", cacheStr},
		{"Last analysis", lastStr},
	})
	return nil
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
