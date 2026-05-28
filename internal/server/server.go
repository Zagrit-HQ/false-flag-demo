// Package server is the FalseFlag control-plane API server. As of
// slice 3 it runs two listeners from one binary: a REST listener on
// cfg.Addr serving oapi-codegen routes, and a ConnectRPC listener on
// cfg.RPCAddr serving the same resource families in protobuf. Both
// listeners share one Store and one set of business rules.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/server/handlers"
	"github.com/depot/falseflag/internal/server/rpc"
	"github.com/depot/falseflag/internal/store"
)

// Server runs the REST and Connect listeners.
type Server struct {
	cfg     appconfig.APIConfig
	log     *slog.Logger
	store   store.Store
	httpSrv *http.Server
	rpcSrv  *http.Server
}

// Deps bundles the constructor's collaborators.
type Deps struct {
	Store store.Store
}

// New constructs the API server. The Store is optional so `make
// api-dev` without compose still produces a usable binary; missing DB
// just disables DB-backed endpoints.
func New(_ context.Context, cfg appconfig.APIConfig, log *slog.Logger, deps Deps) (*Server, error) {
	if log == nil {
		return nil, errors.New("logger is required")
	}
	s := &Server{cfg: cfg, log: log, store: deps.Store}

	s.httpSrv = &http.Server{
		Addr:              cfg.Addr,
		Handler:           s.restRoutes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	rpcProtocols := &http.Protocols{}
	rpcProtocols.SetHTTP1(true)
	rpcProtocols.SetUnencryptedHTTP2(true)
	s.rpcSrv = &http.Server{
		Addr:              cfg.RPCAddr,
		Handler:           s.rpcRoutes(),
		ReadHeaderTimeout: 5 * time.Second,
		Protocols:         rpcProtocols,
	}

	return s, nil
}

// Handler exposes the REST http.Handler for tests.
func (s *Server) Handler() http.Handler { return s.httpSrv.Handler }

// RPCHandler exposes the Connect http.Handler for tests.
func (s *Server) RPCHandler() http.Handler { return s.rpcSrv.Handler }

// Run starts both listeners and blocks until either fails or ctx
// cancels. Migrations run once before either listener binds.
func (s *Server) Run(ctx context.Context) error {
	if s.store != nil {
		if err := s.store.Migrate(ctx, s.log); err != nil {
			return err
		}
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return s.runListener(gctx, s.httpSrv, "rest") })
	g.Go(func() error { return s.runListener(gctx, s.rpcSrv, "rpc") })
	return g.Wait()
}

func (s *Server) runListener(ctx context.Context, srv *http.Server, kind string) error {
	errCh := make(chan error, 1)
	go func() {
		s.log.Info("listener up", "kind", kind, "addr", srv.Addr)
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.log.Info("listener shutting down", "kind", kind)
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) restRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth("healthz"))
	mux.HandleFunc("GET /readyz", s.handleHealth("readyz"))

	api := &handlers.API{Store: s.store, Log: s.log}
	openapi.HandlerFromMux(api, mux)
	// Cap request body size for all REST endpoints. 64 KiB is generous
	// for the largest legitimate payload (a TypeScript flag source up
	// to 32 KiB plus JSON envelope) and prevents accidental or
	// malicious oversize submissions from reaching the compile path.
	return maxBodyBytes(64*1024, mux)
}

// maxBodyBytes wraps h so every request's body is bounded by n bytes.
// Reads past the limit return *http.MaxBytesError; handlers should
// surface those as 413 / InvalidArgument.
func maxBodyBytes(n int64, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, n)
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) rpcRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth("rpc-healthz"))
	rpc.Mount(mux, rpc.Services{Store: s.store, Log: s.log})
	return mux
}

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Probe     string `json:"probe"`
	Timestamp string `json:"timestamp"`
}

func (s *Server) handleHealth(probe string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ok",
			Service:   buildinfo.ServiceName("api"),
			Version:   buildinfo.Version,
			Probe:     probe,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
