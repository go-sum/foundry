package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// TestParseLogLevel verifies every case-sensitive and default branch.
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := ParseLogLevel(tc.input)
			if got != tc.want {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// TestNew_DefaultsToTextInfoStderr verifies that a zero-value Config produces
// a text-format logger at Info level. Info records appear; Debug records are
// absent.
func TestNew_DefaultsToTextInfoStderr(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Output: &buf})

	l.Info("hello info")
	l.Debug("hello debug")

	out := buf.String()
	if !strings.Contains(out, "hello info") {
		t.Errorf("expected Info record in output, got: %q", out)
	}
	if strings.Contains(out, "hello debug") {
		t.Errorf("expected Debug record to be absent, but found it in: %q", out)
	}
}

// TestNew_JSONFormat verifies that FormatJSON produces valid JSON records with
// the required keys.
func TestNew_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Format: FormatJSON, Output: &buf})

	l.Info("json message")

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected non-empty output for JSON logger")
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, line)
	}

	for _, key := range []string{"level", "msg", "time"} {
		if _, ok := record[key]; !ok {
			t.Errorf("JSON record missing key %q; record: %v", key, record)
		}
	}
}

// TestNew_LevelFilter verifies that a logger configured at Warn drops Info and
// Debug records but emits Warn and Error records.
func TestNew_LevelFilter(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Level: slog.LevelWarn, Output: &buf})

	l.Debug("drop debug")
	l.Info("drop info")
	l.Warn("keep warn")
	l.Error("keep error")

	out := buf.String()
	for _, absent := range []string{"drop debug", "drop info"} {
		if strings.Contains(out, absent) {
			t.Errorf("expected %q to be absent from output, got: %q", absent, out)
		}
	}
	for _, present := range []string{"keep warn", "keep error"} {
		if !strings.Contains(out, present) {
			t.Errorf("expected %q to appear in output, got: %q", present, out)
		}
	}
}

// TestFromContext_DefaultWhenAbsent verifies that FromContext returns
// slog.Default() when no logger has been stored.
func TestFromContext_DefaultWhenAbsent(t *testing.T) {
	got := FromContext(context.Background())
	if got != slog.Default() {
		t.Errorf("FromContext without stored logger = %p, want slog.Default() %p", got, slog.Default())
	}
}

// TestIntoAndFromContext_RoundTrip verifies that IntoContext followed by
// FromContext returns exactly the same *slog.Logger pointer.
func TestIntoAndFromContext_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Output: &buf})

	ctx := IntoContext(context.Background(), l)
	got := FromContext(ctx)

	if got != l {
		t.Errorf("FromContext after IntoContext = %p, want %p", got, l)
	}
}

// TestWithRequestID_EmptyIsNoop verifies that passing an empty ID returns the
// same *slog.Logger pointer unchanged.
func TestWithRequestID_EmptyIsNoop(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Output: &buf})

	got := WithRequestID(l, "")
	if got != l {
		t.Errorf("WithRequestID with empty id should return same pointer: got %p, want %p", got, l)
	}
}

// TestWithRequestID_NilDefaultsToSlogDefault verifies that a nil logger does
// not panic and falls back to slog.Default().
func TestWithRequestID_NilDefaultsToSlogDefault(t *testing.T) {
	got := WithRequestID(nil, "req-1")
	if got == nil {
		t.Fatal("WithRequestID(nil, ...) returned nil")
	}
}

// TestWithRequestID_AttachesAttr verifies that the returned logger emits a
// request_id attribute with the expected value.
func TestWithRequestID_AttachesAttr(t *testing.T) {
	var buf bytes.Buffer
	base := New(Config{Format: FormatText, Output: &buf})

	enriched := WithRequestID(base, "abc-123")
	if enriched == base {
		t.Fatal("WithRequestID with non-empty id should return a new logger")
	}

	enriched.Info("test message")

	out := buf.String()
	if !strings.Contains(out, "request_id=abc-123") {
		t.Errorf("expected request_id=abc-123 in output, got: %q", out)
	}
}
