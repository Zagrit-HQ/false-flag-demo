// Command falseflag-api is the FalseFlag control-plane HTTP server.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/logging"
	"github.com/depot/falseflag/internal/server"
	"github.com/depot/falseflag/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	os.Exit(buildinfo.WithGracefulShutdown("api", run))
}

func healthcheck() int {
	addr := os.Getenv("FALSEFLAG_API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	url := "http://" + healthHost(addr) + "/healthz"

	client := http.Client{Timeout: 2 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "healthcheck returned %s\n", res.Status)
		return 1
	}
	return 0
}

func healthHost(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
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
