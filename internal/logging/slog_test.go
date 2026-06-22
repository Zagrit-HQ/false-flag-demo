package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	cases := map[string]slog.Level{
		"debug":    slog.LevelDebug,
		"DEBUG":    slog.LevelDebug,
		"info":     slog.LevelInfo,
		"  warn  ": slog.LevelWarn,
		"warning":  slog.LevelWarn,
		"error":    slog.LevelError,
		"":         slog.LevelInfo,
		"unknown":  slog.LevelInfo,
	}
	for raw, want := range cases {
		if got := ParseLevel(raw); got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestNewWithOptions_TagsServiceAndVersion(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := NewWithOptions("api", Options{Writer: &buf})
	log.Info("hello")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log output was not JSON: %v\nraw: %s", err, buf.String())
	}
	if got, want := record["service.name"], "falseflag-api"; got != want {
		t.Errorf("service.name = %v, want %v", got, want)
	}
	if _, ok := record["service.version"]; !ok {
		t.Errorf("expected service.version to be set; got %v", record)
	}
	if got, want := record["msg"], "hello"; got != want {
		t.Errorf("msg = %v, want %v", got, want)
	}
}

func TestNewWithOptions_LevelGate(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	warn := slog.LevelWarn
	log := NewWithOptions("api", Options{Writer: &buf, Level: &warn})

	log.Info("filtered")
	log.Warn("kept")

	if bytes.Contains(buf.Bytes(), []byte("filtered")) {
		t.Errorf("info message leaked through warn-level handler: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("kept")) {
		t.Errorf("expected warn message in output, got: %s", buf.String())
	}
}

func TestNew_RespectsLogLevelEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "error")
	var buf bytes.Buffer
	log := NewWithOptions("api", Options{Writer: &buf})

	log.Warn("should be dropped")
	log.Error("should pass")

	if bytes.Contains(buf.Bytes(), []byte("should be dropped")) {
		t.Errorf("warn leaked through error gate")
	}
	if !bytes.Contains(buf.Bytes(), []byte("should pass")) {
		t.Errorf("error did not pass through")
	}
}
