package ui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// ── ParseFormat ───────────────────────────────────────────────────────────────

func TestParseFormat_JSON(t *testing.T) {
	if got := ParseFormat("json"); got != FormatJSON {
		t.Errorf("want FormatJSON, got %q", got)
	}
}

func TestParseFormat_Human(t *testing.T) {
	if got := ParseFormat("human"); got != FormatHuman {
		t.Errorf("want FormatHuman, got %q", got)
	}
}

func TestParseFormat_EmptyDefaultsToHuman(t *testing.T) {
	if got := ParseFormat(""); got != FormatHuman {
		t.Errorf("empty string: want FormatHuman, got %q", got)
	}
}

func TestParseFormat_UnknownDefaultsToHuman(t *testing.T) {
	if got := ParseFormat("yaml"); got != FormatHuman {
		t.Errorf("unknown format: want FormatHuman, got %q", got)
	}
}

// ── JSON ──────────────────────────────────────────────────────────────────────

func TestJSON_ValidOutput(t *testing.T) {
	var buf bytes.Buffer
	v := map[string]any{"name": "test", "count": 42}
	if err := JSON(&buf, v); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	// Must be valid JSON
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if decoded["name"] != "test" {
		t.Errorf("decoded name: got %v", decoded["name"])
	}
}

func TestJSON_Indented(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, map[string]any{"x": 1}); err != nil {
		t.Fatal(err)
	}
	// Indented JSON should contain newlines
	if !strings.Contains(buf.String(), "\n") {
		t.Errorf("JSON output should be indented, got: %s", buf.String())
	}
}

func TestJSON_Slice(t *testing.T) {
	var buf bytes.Buffer
	items := []string{"a", "b", "c"}
	if err := JSON(&buf, items); err != nil {
		t.Fatal(err)
	}
	var decoded []string
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(decoded) != 3 || decoded[0] != "a" {
		t.Errorf("decoded: %v", decoded)
	}
}

// ── Table ─────────────────────────────────────────────────────────────────────

func TestTable_ContainsHeaders(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []string{"NAME", "VALUE"}, [][]string{{"foo", "bar"}})
	out := buf.String()
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "VALUE") {
		t.Errorf("should contain headers, got:\n%s", out)
	}
}

func TestTable_ContainsRows(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []string{"KEY", "VAL"}, [][]string{
		{"alpha", "1"},
		{"beta", "2"},
	})
	out := buf.String()
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Errorf("should contain row data, got:\n%s", out)
	}
}

func TestTable_Separator(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []string{"NAME"}, [][]string{{"row"}})
	out := buf.String()
	// Should have a separator line of dashes
	if !strings.Contains(out, "----") {
		t.Errorf("should have separator dashes, got:\n%s", out)
	}
}

func TestTable_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []string{"FIELD"}, nil)
	out := buf.String()
	// Headers and separator should still appear
	if !strings.Contains(out, "FIELD") {
		t.Errorf("empty rows: should still show headers, got:\n%s", out)
	}
}

func TestTable_MultiColumn(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []string{"A", "B", "C"}, [][]string{{"x", "y", "z"}})
	out := buf.String()
	for _, want := range []string{"A", "B", "C", "x", "y", "z"} {
		if !strings.Contains(out, want) {
			t.Errorf("should contain %q, got:\n%s", want, out)
		}
	}
}
