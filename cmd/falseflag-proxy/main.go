// Command falseflag-proxy is the FalseFlag evaluation proxy.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/logging"
	"github.com/depot/falseflag/internal/proxy"
)

func main() {
	os.Exit(buildinfo.WithGracefulShutdown("proxy", run))
}

func run(ctx context.Context) error {
	log := logging.New("proxy")

	cfg, err := appconfig.LoadProxy()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	p, err := proxy.New(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("initializing proxy: %w", err)
	}

	log.Info("starting falseflag-proxy",
		"version", buildinfo.Version,
		"commit", buildinfo.Commit,
		"addr", cfg.Addr,
	)
	return p.Run(ctx)
}
