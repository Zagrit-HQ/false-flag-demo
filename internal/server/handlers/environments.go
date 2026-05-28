package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
)

func (a *API) ListEnvironments(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	rows, err := a.Store.ListEnvironmentsByProject(r.Context(), proj.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]openapi.Environment, 0, len(rows))
	for _, e := range rows {
		items = append(items, environmentToAPI(e))
	}
	writeJSON(w, http.StatusOK, openapi.EnvironmentList{Items: items})
}

func (a *API) CreateEnvironment(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	var req openapi.CreateEnvironmentRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	var env store.Environment
	err = a.Store.WithAudit(r.Context(), store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "create_environment",
		Actor:     actorFromRequest(r),
		Payload:   mustMarshal(map[string]any{"slug": req.Slug}),
	}, func(tx store.Tx) error {
		e, err := tx.CreateEnvironment(r.Context(), proj.ID, req.Slug, req.Name)
		if err != nil {
			return err
		}
		env = e
		return nil
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, environmentToAPI(env))
}

func (a *API) GetEnvironment(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, envSlug openapi.EnvironmentSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	env, err := a.Store.GetEnvironmentBySlug(r.Context(), proj.ID, envSlug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, environmentToAPI(env))
}
