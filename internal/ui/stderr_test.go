package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// captureStderr temporarily redirects os.Stderr to a pipe and returns
// everything written to it after calling fn.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = orig

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// ── Step ──────────────────────────────────────────────────────────────────────

func TestStep_WritesToStderr(t *testing.T) {
	out := captureStderr(t, func() {
		Step("doing the thing")
	})
	if !strings.Contains(out, "doing the thing") {
		t.Errorf("Step: want 'doing the thing' in stderr, got: %q", out)
	}
}

func TestStep_PrefixArrow(t *testing.T) {
	out := captureStderr(t, func() {
		Step("hello")
	})
	if !strings.Contains(out, "→") {
		t.Errorf("Step: want arrow prefix, got: %q", out)
	}
}

// ── Success ───────────────────────────────────────────────────────────────────

func TestSuccess_WritesToStderr(t *testing.T) {
	out := captureStderr(t, func() {
		Success("done with %d items", 42)
	})
	if !strings.Contains(out, "done with 42 items") {
		t.Errorf("Success: want message in stderr, got: %q", out)
	}
}

func TestSuccess_PrefixCheckmark(t *testing.T) {
	out := captureStderr(t, func() {
		Success("ok")
	})
	if !strings.Contains(out, "✓") {
		t.Errorf("Success: want checkmark prefix, got: %q", out)
	}
}

// ── Warn ──────────────────────────────────────────────────────────────────────

func TestWarn_WritesToStderr(t *testing.T) {
	out := captureStderr(t, func() {
		Warn("something is wrong: %s", "disk full")
	})
	if !strings.Contains(out, "something is wrong: disk full") {
		t.Errorf("Warn: want message in stderr, got: %q", out)
	}
}

func TestWarn_PrefixWarning(t *testing.T) {
	out := captureStderr(t, func() {
		Warn("bad thing")
	})
	if !strings.Contains(out, "warning:") {
		t.Errorf("Warn: want 'warning:' prefix, got: %q", out)
	}
}

// ── Spinner ───────────────────────────────────────────────────────────────────

func TestSpinner_StartStop(t *testing.T) {
	// Just verify Start/Stop don't panic and the line is cleared on stop.
	out := captureStderr(t, func() {
		s := Start("loading…")
		time.Sleep(100 * time.Millisecond) // let it tick at least once
		s.Stop()
	})
	// After Stop, the spinner clears its line with spaces; the message
	// may or may not be in the captured output depending on timing, but
	// we should at minimum not see a partial braille character stuck on
	// the terminal (the clear sequence is a carriage return + spaces).
	if strings.Contains(out, "loading…") && !strings.Contains(out, "\r") {
		t.Errorf("Spinner: expected carriage-return clear sequence, got: %q", out)
	}
}

func TestSpinner_StopClearsLine(t *testing.T) {
	out := captureStderr(t, func() {
		s := Start("working")
		time.Sleep(200 * time.Millisecond)
		s.Stop()
	})
	// Stop writes "\r%-70s\r" to clear the line.
	if !strings.Contains(out, "\r") {
		t.Errorf("Spinner.Stop: expected carriage return in output, got: %q", out)
	}
}
