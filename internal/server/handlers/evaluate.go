package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/eval"
	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
)

func (a *API) EvaluateFlag(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, key openapi.FlagKey) {
	if !a.requireStore(w) {
		return
	}
	compiled, version, ctx, ok := a.loadEvaluation(w, r, slug, key)
	if !ok {
		return
	}
	d, err := eval.Evaluate(compiled, ctx, version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, decisionToAPI(d))
}

func (a *API) EvaluateFlagWithTrace(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, key openapi.FlagKey) {
	if !a.requireStore(w) {
		return
	}
	compiled, version, ctx, ok := a.loadEvaluation(w, r, slug, key)
	if !ok {
		return
	}
	d, trace, err := eval.EvaluateWithTrace(compiled, ctx, version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, openapi.EvaluateTraceResponse{
		Decision: decisionToAPI(d),
		Trace:    traceToAPI(trace),
	})
}

// loadEvaluation is the shared front matter for /evaluate and
// /evaluate-trace: resolve project, flag, latest version, decode the
// request context, and rehydrate the compiled IR. Returns ok=false
// after writing the HTTP error response itself.
func (a *API) loadEvaluation(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, key openapi.FlagKey) (*config.Compiled, int, map[string]any, bool) {
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return nil, 0, nil, false
	}
	flag, err := a.Store.GetFlagByKey(r.Context(), proj.ID, key)
	if err != nil {
		notFoundOrError(w, err)
		return nil, 0, nil, false
	}
	version, err := a.Store.GetLatestFlagVersion(r.Context(), flag.ID)
	if err != nil {
		notFoundOrError(w, err)
		return nil, 0, nil, false
	}
	var req openapi.EvaluationRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, err)
		return nil, 0, nil, false
	}
	compiled, err := config.Compile(config.Strategy(version.Strategy), version.Compiled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("rehydrating IR: %w", err))
		return nil, 0, nil, false
	}
	return compiled, version.Version, req.Context, true
}

// guard against unused-import (context for future deadline wiring).
var _ = errors.Is
var _ = context.Canceled
var _ store.Project
