// Package logging is the shared log/slog factory for every FalseFlag Go
// binary. We use the standard library handler so we don't pull in zap or
// zerolog; the chosen stack mandates log/slog.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/depot/falseflag/internal/buildinfo"
)

// Options control the slog handler. All fields are optional.
type Options struct {
	// Level overrides LOG_LEVEL when non-nil. Useful for tests.
	Level *slog.Level
	// Writer overrides os.Stdout when non-nil. Useful for tests.
	Writer io.Writer
}

// New returns a JSON slog.Logger that tags every record with the FalseFlag
// service name (e.g. "falseflag-api") and the binary's build version. Level
// defaults to info and can be overridden with LOG_LEVEL=debug|info|warn|error.
func New(serviceSuffix string) *slog.Logger {
	return NewWithOptions(serviceSuffix, Options{})
}

// NewWithOptions is the same as New but accepts explicit overrides for the
// level and writer. Production code should prefer New.
func NewWithOptions(serviceSuffix string, opts Options) *slog.Logger {
	level := slog.LevelInfo
	if opts.Level != nil {
		level = *opts.Level
	} else if raw := os.Getenv("LOG_LEVEL"); raw != "" {
		level = ParseLevel(raw)
	}

	writer := io.Writer(os.Stdout)
	if opts.Writer != nil {
		writer = opts.Writer
	}

	h := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	return slog.New(h).With(
		"service.name", buildinfo.ServiceName(serviceSuffix),
		"service.version", buildinfo.Version,
	)
}

// ParseLevel turns LOG_LEVEL strings into slog.Level. Unknown values
// default to info — a typo shouldn't silence logs.
func ParseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
