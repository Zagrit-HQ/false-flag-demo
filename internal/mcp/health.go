package mcp

import (
	"encoding/json"
	"net/http"

	"github.com/depot/falseflag/internal/buildinfo"
)

// healthHandler returns an http.Handler that serves /healthz with the
// same JSON shape the proxy uses, so compose healthchecks and CI
// scripts can probe FalseFlag binaries uniformly.
func healthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": buildinfo.ServiceName("mcp"),
			"version": buildinfo.Version,
			"probe":   "liveness",
		})
	})
	return mux
}
