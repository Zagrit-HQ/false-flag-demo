// Command falseflag-api is the FalseFlag control-plane HTTP server.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/healthcheck"
	"github.com/depot/falseflag/internal/logging"
	"github.com/depot/falseflag/internal/server"
	"github.com/depot/falseflag/internal/store"
)

func main() {
	if healthcheck.RunFromArgs(os.Args) {
		return
	}
	os.Exit(buildinfo.WithGracefulShutdown("api", run))
}

func run(ctx context.Context) error {
	log := logging.New("api")

	cfg, err := appconfig.LoadAPI()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	deps := server.Deps{}
	if cfg.DatabaseURL != "" {
		s, err := store.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer s.Close()
		deps.Store = s
		log.Info("store ready", "database", redact(cfg.DatabaseURL))
	} else {
		log.Warn("no FALSEFLAG_DATABASE_URL set; DB-backed endpoints disabled")
	}

	srv, err := server.New(ctx, cfg, log, deps)
	if err != nil {
		return fmt.Errorf("initializing server: %w", err)
	}

	log.Info("starting falseflag-api",
		"build-buster", "afdsafdsa",
		"version", buildinfo.Version,
		"commit", buildinfo.Commit,
		"addr", cfg.Addr,
	)
	return srv.Run(ctx)
}

// redact removes the password component of a database URL for safe
// logging. It is intentionally approximate; the goal is to keep
// secrets out of slog output, not to be cryptographically careful.
func redact(url string) string {
	at := -1
	for i := 0; i < len(url); i++ {
		if url[i] == '@' {
			at = i
			break
		}
	}
	if at < 0 {
		return url
	}
	return "postgres://***@" + url[at+1:]
}
