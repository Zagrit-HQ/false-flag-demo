// Package buildinfo exposes static identity and version information about
// the FalseFlag binaries, plus a small graceful-shutdown helper that every
// cmd/*/main.go wraps its run function in.
package buildinfo

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// Name is the lowercase product name used in service names, container image
// tags, and OpenTelemetry resource attributes.
const Name = "falseflag"

// Version and Commit are populated at build time via -ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
)

// ServiceName returns the canonical service name for a given binary suffix.
// ServiceName("") returns the bare product name.
func ServiceName(suffix string) string {
	if suffix == "" {
		return Name
	}
	return Name + "-" + suffix
}

// WithGracefulShutdown runs fn with a context that is cancelled on
// SIGINT/SIGTERM. It returns the appropriate process exit code: 0 on a
// clean shutdown, 1 if fn returns a non-context error.
//
// This is the shape every cmd/falseflag-*/main.go uses; it matches the
// pattern in project-depot/registry/cmd/registry/main.go.
func WithGracefulShutdown(serviceSuffix string, fn func(context.Context) error) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		select {
		case sig := <-sigChan:
			slog.Info("received signal, shutting down", "signal", sig.String(), "service", ServiceName(serviceSuffix))
			cancel()
		case <-ctx.Done():
		}
	}()

	if err := fn(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("service exited with error", "service", ServiceName(serviceSuffix), "error", err)
		return 1
	}
	return 0
}
