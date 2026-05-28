// Package handlers implements the OpenAPI-generated ServerInterface
// for the FalseFlag control plane. Each resource family lives in its
// own file; handlers.go owns the API struct, the lifecycle helpers
// (requireStore, decodeJSON, writeJSON, writeError, error translation)
// and the shared utility helpers (actorFromRequest, derefString).
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
)

// API holds collaborators every handler needs.
type API struct {
	Store store.Store
	Log   *slog.Logger
}

// Compile-time check: API satisfies the generated ServerInterface.
var _ openapi.ServerInterface = (*API)(nil)

// ---- Health --------------------------------------------------------

func (a *API) GetHealth(w http.ResponseWriter, _ *http.Request) {
	now := time.Now().UTC()
	probe := "v1.health"
	writeJSON(w, http.StatusOK, openapi.HealthResponse{
		Status:    openapi.HealthResponseStatus("ok"),
		Service:   buildinfo.ServiceName("api"),
		Version:   buildinfo.Version,
		Probe:     &probe,
		Timestamp: &now,
	})
}

// ---- HTTP helpers --------------------------------------------------

func (a *API) requireStore(w http.ResponseWriter) bool {
	if a.Store == nil {
		writeError(w, http.StatusServiceUnavailable,
			errors.New("DB-backed endpoints disabled: set FALSEFLAG_DATABASE_URL"))
		return false
	}
	return true
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, err error) {
	resp := openapi.Error{Error: err.Error()}
	if det := errors.Unwrap(err); det != nil {
		s := det.Error()
		resp.Details = &s
	}
	writeJSON(w, status, resp)
}

// notFoundOrError translates store sentinels into HTTP codes. ErrNotFound
// becomes 404, anything else 500.
func notFoundOrError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeError(w, http.StatusInternalServerError, err)
}

// writeStoreErr is the catch-all error translator for mutation handlers:
// not-found → 404, unique/FK violation → 409, anything else → 500.
func writeStoreErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case store.IsConflict(err):
		writeError(w, http.StatusConflict, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func badRequest(w http.ResponseWriter, err error) {
	writeError(w, http.StatusBadRequest, err)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// actorFromRequest returns the X-Actor header value or "" when absent.
// FalseFlag has no real auth in slice 3; this is a demo-only attribution
// signal recorded against audit events.
func actorFromRequest(r *http.Request) string {
	return r.Header.Get("X-Actor")
}
