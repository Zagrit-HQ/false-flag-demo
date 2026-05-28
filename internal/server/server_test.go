package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/logging"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	log := logging.New("api")
	srv, err := New(context.Background(), appconfig.APIConfig{Addr: ":0", RPCAddr: ":0"}, log, Deps{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return srv
}

func TestServer_HealthRoutes(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, route := range []string{"/healthz", "/readyz", "/v1/health"} {
		t.Run(route, func(t *testing.T) {
			res, err := http.Get(ts.URL + route)
			if err != nil {
				t.Fatalf("GET %s: %v", route, err)
			}
			defer func() { _ = res.Body.Close() }()

			if res.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.StatusCode)
			}
			body, _ := io.ReadAll(res.Body)
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("body not JSON: %v\nraw: %s", err, body)
			}
			if payload["status"] != "ok" {
				t.Errorf("status field = %q, want ok", payload["status"])
			}
			if payload["service"] != "falseflag-api" {
				t.Errorf("service field = %q, want falseflag-api", payload["service"])
			}
			if payload["timestamp"] == "" {
				t.Errorf("timestamp missing")
			}
		})
	}
}

func TestServer_RunHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Run did not exit after context cancellation")
	}
}

func TestServer_RequiresLogger(t *testing.T) {
	t.Parallel()
	if _, err := New(context.Background(), appconfig.APIConfig{Addr: ":0"}, nil, Deps{}); err == nil {
		t.Fatalf("expected error when logger is nil")
	}
}
