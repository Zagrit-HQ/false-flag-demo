// Package mcp implements the FalseFlag agent-facing MCP (Model
// Context Protocol) server. It wraps the slice-3 Connect control
// plane in tool calls an LLM agent can invoke: list_projects,
// list_flags, get_flag, validate_config, explain_evaluation,
// search_audit_log.
//
// Quality bar: demo-quality. No bearer-token auth; X-Actor stamping
// is attribution only.
package mcp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/logging"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

// Run is the entrypoint wired from cmd/falseflag-mcp/main.go via
// buildinfo.WithGracefulShutdown. It boots two listeners under one
// errgroup: the MCP Streamable HTTP surface on cfg.Addr and a
// dedicated /healthz on cfg.HealthAddr.
func Run(ctx context.Context) error {
	log := logging.New("mcp")
	cfg, err := appconfig.LoadMCP()
	if err != nil {
		return err
	}
	client := clientapi.New(cfg.APIBaseURL, cfg.Actor)
	srv := newServer()
	RegisterTools(srv, client)

	mcpHandler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return srv
	}, nil)
	mcpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mcpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	healthSrv := &http.Server{
		Addr:              cfg.HealthAddr,
		Handler:           healthHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("falseflag-mcp starting",
		"addr", cfg.Addr,
		"health_addr", cfg.HealthAddr,
		"api_base_url", cfg.APIBaseURL,
		"actor", cfg.Actor,
		"tools", ToolNames,
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return runListener(gctx, log, mcpSrv, "mcp") })
	g.Go(func() error { return runListener(gctx, log, healthSrv, "health") })
	return g.Wait()
}

func runListener(ctx context.Context, log *slog.Logger, srv *http.Server, kind string) error {
	errCh := make(chan error, 1)
	go func() {
		log.Info("listener up", "kind", kind, "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}
