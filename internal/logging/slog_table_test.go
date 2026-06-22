package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel_Table(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"  debug  ", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"", slog.LevelInfo},
		{"notalevel", slog.LevelInfo},
		{"5", slog.LevelInfo},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			if got := ParseLevel(tc.raw); got != tc.want {
				t.Errorf("ParseLevel(%q) = %v want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestNew_DefaultsToInfo(t *testing.T) {
	t.Parallel()
	logger := New("api")
	if logger == nil {
		t.Fatal("New returned nil")
	}
}

func TestNewWithOptions_LevelOverride(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	lvl := slog.LevelWarn
	logger := NewWithOptions("api", Options{Level: &lvl, Writer: &buf})
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// Debug and Info should be filtered out; Warn and Error pass.
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), buf.String())
	}
	for i, line := range lines {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("decode line %d: %v", i, err)
		}
		if rec["service.name"] != "falseflag-api" {
			t.Errorf("service.name = %v", rec["service.name"])
		}
	}
}

func TestNewWithOptions_LevelsFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		level slog.Level
		want  int // number of lines from a {Debug,Info,Warn,Error} burst
	}{
		{slog.LevelDebug, 4},
		{slog.LevelInfo, 3},
		{slog.LevelWarn, 2},
		{slog.LevelError, 1},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.level.String(), func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			l := NewWithOptions("api", Options{Level: &tc.level, Writer: &buf})
			l.Debug("d")
			l.Info("i")
			l.Warn("w")
			l.Error("e")
			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			if buf.Len() == 0 {
				lines = nil
			}
			if len(lines) != tc.want {
				t.Errorf("at %v: got %d lines, want %d:\n%s", tc.level, len(lines), tc.want, buf.String())
			}
		})
	}
}

func TestNewWithOptions_WriterCaptures(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := NewWithOptions("proxy", Options{Writer: &buf})
	l.Info("hello", "k", "v")
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec["msg"] != "hello" || rec["k"] != "v" {
		t.Errorf("rec = %+v", rec)
	}
	if rec["service.name"] != "falseflag-proxy" {
		t.Errorf("service.name = %v", rec["service.name"])
	}
}

func TestNewWithOptions_ServiceSuffix(t *testing.T) {
	t.Parallel()
	suffixes := []string{"", "api", "proxy", "operator", "mcp"}
	for _, s := range suffixes {
		s := s
		t.Run("suffix="+s, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			l := NewWithOptions(s, Options{Writer: &buf})
			l.Info("x")
			var rec map[string]any
			if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
				t.Fatalf("decode: %v", err)
			}
			want := "falseflag"
			if s != "" {
				want = "falseflag-" + s
			}
			if rec["service.name"] != want {
				t.Errorf("service.name = %v want %v", rec["service.name"], want)
			}
		})
	}
}
