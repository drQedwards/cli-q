package blastradius

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// ── normalise ─────────────────────────────────────────────────────────────────

func TestNormalise_RelativePath(t *testing.T) {
	got := normalise("/repo", "internal/api/client.go")
	want := "internal/api/client.go"
	if got != want {
		t.Errorf("normalise relative: got %q, want %q", got, want)
	}
}

func TestNormalise_AbsolutePath(t *testing.T) {
	got := normalise("/repo", "/repo/internal/api/client.go")
	want := "internal/api/client.go"
	if got != want {
		t.Errorf("normalise absolute: got %q, want %q", got, want)
	}
}

func TestNormalise_DotSlashPrefix(t *testing.T) {
	got := normalise("/repo", "./internal/api/client.go")
	want := "internal/api/client.go"
	if got != want {
		t.Errorf("normalise dot-slash: got %q, want %q", got, want)
	}
}

func TestNormalise_SlashSeparators(t *testing.T) {
	got := normalise(".", "internal/api/client.go")
	if strings.Contains(got, "\\") {
		t.Errorf("normalise should use forward slashes: %q", got)
	}
}

// ── printResults ──────────────────────────────────────────────────────────────

func TestPrintResults_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := printResults(&buf, "cmd/main.go", nil, "human"); err != nil {
		t.Fatalf("printResults empty: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No files") {
		t.Errorf("empty results: should say 'No files':\n%s", out)
	}
	if !strings.Contains(out, "cmd/main.go") {
		t.Errorf("should mention target:\n%s", out)
	}
}

func TestPrintResults_Human(t *testing.T) {
	results := []Result{
		{File: "internal/auth/handler.go", Depth: 1},
		{File: "cmd/main.go", Depth: 2},
	}
	var buf bytes.Buffer
	if err := printResults(&buf, "internal/api/client.go", results, "human"); err != nil {
		t.Fatalf("printResults human: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"internal/auth/handler.go", "cmd/main.go", "2 file(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("should contain %q:\n%s", want, out)
		}
	}
}

func TestPrintResults_JSON(t *testing.T) {
	results := []Result{
		{File: "internal/auth/handler.go", Depth: 1},
	}
	var buf bytes.Buffer
	if err := printResults(&buf, "internal/api/client.go", results, "json"); err != nil {
		t.Fatalf("printResults json: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if decoded["target"] != "internal/api/client.go" {
		t.Errorf("target: got %v", decoded["target"])
	}
	affected, ok := decoded["affected"].([]any)
	if !ok || len(affected) != 1 {
		t.Errorf("affected: want 1 item, got %v", decoded["affected"])
	}
}

func TestPrintResults_HumanShowsHops(t *testing.T) {
	results := []Result{
		{File: "a.go", Depth: 3},
	}
	var buf bytes.Buffer
	if err := printResults(&buf, "b.go", results, "human"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "3") {
		t.Errorf("should show depth/hops:\n%s", out)
	}
}
