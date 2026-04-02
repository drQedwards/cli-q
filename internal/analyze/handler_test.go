package analyze

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/supermodeltools/cli/internal/api"
	"github.com/supermodeltools/cli/internal/ui"
)

func TestPrintSummary_JSON(t *testing.T) {
	g := &api.Graph{
		Metadata: map[string]any{"repoId": "repo-abc"},
		Nodes: []api.Node{
			{ID: "f1", Labels: []string{"File"}, Properties: map[string]any{"path": "main.go"}},
			{ID: "f2", Labels: []string{"File"}, Properties: map[string]any{"path": "handler.go"}},
			{ID: "fn1", Labels: []string{"Function"}, Properties: map[string]any{"name": "main"}},
		},
		Relationships: []api.Relationship{
			{ID: "r1", Type: "CALLS", StartNode: "fn1", EndNode: "fn1"},
		},
	}
	var buf bytes.Buffer
	if err := printSummary(&buf, g, ui.FormatJSON); err != nil {
		t.Fatalf("printSummary JSON: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if decoded["repo_id"] != "repo-abc" {
		t.Errorf("repo_id: got %v", decoded["repo_id"])
	}
	if decoded["files"].(float64) != 2 {
		t.Errorf("files: want 2, got %v", decoded["files"])
	}
	if decoded["functions"].(float64) != 1 {
		t.Errorf("functions: want 1, got %v", decoded["functions"])
	}
	if decoded["relationships"].(float64) != 1 {
		t.Errorf("relationships: want 1, got %v", decoded["relationships"])
	}
}

func TestPrintSummary_Human(t *testing.T) {
	g := &api.Graph{
		Metadata: map[string]any{"repoId": "my-repo"},
		Nodes: []api.Node{
			{ID: "f1", Labels: []string{"File"}, Properties: map[string]any{"path": "a.go"}},
			{ID: "fn1", Labels: []string{"Function"}, Properties: map[string]any{"name": "Foo"}},
			{ID: "fn2", Labels: []string{"Function"}, Properties: map[string]any{"name": "Bar"}},
		},
		Relationships: []api.Relationship{
			{ID: "r1", Type: "CALLS", StartNode: "fn1", EndNode: "fn2"},
			{ID: "r2", Type: "CALLS", StartNode: "fn2", EndNode: "fn1"},
		},
	}
	var buf bytes.Buffer
	if err := printSummary(&buf, g, ui.FormatHuman); err != nil {
		t.Fatalf("printSummary human: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"my-repo", "1", "2"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}
}

func TestPrintSummary_EmptyGraph(t *testing.T) {
	g := &api.Graph{}
	var buf bytes.Buffer
	if err := printSummary(&buf, g, ui.FormatHuman); err != nil {
		t.Fatalf("printSummary empty graph: %v", err)
	}
	out := buf.String()
	// Should contain zeros
	if !strings.Contains(out, "0") {
		t.Errorf("empty graph: expected zeros in output:\n%s", out)
	}
}
