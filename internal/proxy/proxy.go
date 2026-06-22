// Package proxy is the FalseFlag evaluation proxy HTTP server. It
// wraps internal/sdkgo to expose a small REST surface
// (POST /v1/evaluate, GET /healthz) so consumers that can't run an
// SDK directly still get local snapshot evaluation behind a
// long-lived edge process.
package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/eval"
	"github.com/depot/falseflag/internal/sdkgo"
)

// Evaluator is the surface the proxy needs from a SDK client. Defined
// as an interface so tests can substitute a fake without spinning up
// an httptest snapshot server.
type Evaluator interface {
	Evaluate(key string, ctx sdkgo.EvalContext) eval.Decision
	Snapshot() *sdkgo.Snapshot
}

// Proxy holds the configured HTTP server and the embedded SDK
// client.
type Proxy struct {
	cfg       appconfig.ProxyConfig
	log       *slog.Logger
	srv       *http.Server
	evaluator Evaluator
	sdkClient *sdkgo.Client // nil when Evaluator was injected for tests
}

// New constructs a Proxy. The SDK client is created but not started;
// call Run (or Start in tests) to begin polling.
func New(_ context.Context, cfg appconfig.ProxyConfig, log *slog.Logger) (*Proxy, error) {
	if log == nil {
		return nil, errors.New("logger is required")
	}
	p := &Proxy{cfg: cfg, log: log}
	if cfg.ProjectSlug != "" && cfg.APIBaseURL != "" {
		client, err := sdkgo.NewClient(sdkgo.Options{
			BaseURL:     cfg.APIBaseURL,
			ProjectSlug: cfg.ProjectSlug,
			Logger:      log,
		})
		if err != nil {
			return nil, err
		}
		p.sdkClient = client
		p.evaluator = client
	}
	p.srv = &http.Server{
		Addr:              cfg.Addr,
		Handler:           p.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return p, nil
}

// SetEvaluator overrides the embedded evaluator. Test-only.
func (p *Proxy) SetEvaluator(e Evaluator) { p.evaluator = e }

// Handler exposes the routes for httptest use.
func (p *Proxy) Handler() http.Handler { return p.srv.Handler }

// Run boots the embedded SDK client (if any), starts the HTTP server,
// and blocks until ctx is canceled.
func (p *Proxy) Run(ctx context.Context) error {
	if p.sdkClient != nil {
		if err := p.sdkClient.Start(ctx); err != nil {
			p.log.Warn("initial snapshot poll failed; proxy will return 503 until next poll",
				"err", err,
			)
		}
		defer p.sdkClient.Stop()
	}

	errCh := make(chan error, 1)
	go func() {
		p.log.Info("proxy listening",
			"addr", p.cfg.Addr,
			"api", p.cfg.APIBaseURL,
			"project", p.cfg.ProjectSlug,
		)
		err := p.srv.ListenAndServe()
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
		p.log.Info("proxy shutting down")
		return p.srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (p *Proxy) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", p.handleHealth)
	mux.HandleFunc("GET /readyz", p.handleReady)
	mux.HandleFunc("POST /v1/evaluate", p.handleEvaluate)
	mux.HandleFunc("GET /v1/snapshot", p.handleSnapshotInfo)
	return mux
}

// handleHealth is the liveness probe: returns 200 once the binary is
// up. The slice-1 contract is preserved (status=ok / service=falseflag-proxy).
func (p *Proxy) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": buildinfo.ServiceName("proxy"),
		"version": buildinfo.Version,
	})
}

// handleReady is the readiness probe: returns 200 only when a snapshot
// is loaded and any configured ready flags are present (or no project
// is configured, in which case the proxy is idle-ready). Returns 503
// with a structured body when the proxy is expected to serve
// evaluations but hasn't loaded its first usable snapshot yet.
func (p *Proxy) handleReady(w http.ResponseWriter, _ *http.Request) {
	status := "ok"
	var missing []string
	if p.evaluator != nil && p.evaluator.Snapshot() == nil {
		status = "starting"
	} else if p.evaluator != nil {
		snap := p.evaluator.Snapshot()
		for _, key := range p.cfg.ReadyFlags {
			if _, ok := snap.Flags[key]; !ok {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			status = "missing_required_flags"
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":                 status,
		"project":                p.cfg.ProjectSlug,
		"required_ready_flags":   p.cfg.ReadyFlags,
		"missing_required_flags": missing,
	})
}

// EvaluateRequest is the body shape accepted by POST /v1/evaluate.
type EvaluateRequest struct {
	Key          string         `json:"key"`
	DefaultValue any            `json:"default_value,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
}

func (p *Proxy) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if p.evaluator == nil {
		writeJSONError(w, http.StatusServiceUnavailable,
			"no_project_configured",
			"proxy was started without FALSEFLAG_PROXY_PROJECT_SLUG")
		return
	}
	var req EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.Key == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_key", "key is required")
		return
	}
	d := p.evaluator.Evaluate(req.Key, req.Context)
	// Substitute default when value is nil — mirrors the OpenFeature-
	// shaped client behavior. This is intentionally permissive: the
	// caller controls whether the default is a bool, string, number,
	// or object.
	if d.Value == nil && req.DefaultValue != nil {
		d.Value = req.DefaultValue
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(d)
}

// SnapshotInfo describes the loaded snapshot without leaking the
// per-flag IR.
type SnapshotInfo struct {
	ID          string    `json:"id"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	FlagCount   int       `json:"flag_count"`
	ProjectSlug string    `json:"project_slug"`
}

func (p *Proxy) handleSnapshotInfo(w http.ResponseWriter, _ *http.Request) {
	if p.evaluator == nil {
		writeJSONError(w, http.StatusServiceUnavailable,
			"no_project_configured", "")
		return
	}
	snap := p.evaluator.Snapshot()
	if snap == nil {
		writeJSONError(w, http.StatusServiceUnavailable,
			"no_snapshot", "first poll has not completed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(SnapshotInfo{
		ID:          snap.ID,
		Version:     snap.Version,
		CreatedAt:   snap.CreatedAt,
		FlagCount:   len(snap.Flags),
		ProjectSlug: p.cfg.ProjectSlug,
	})
}

func writeJSONError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": msg,
	})
}
