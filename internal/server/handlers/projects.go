package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
)

func (a *API) ListProjects(w http.ResponseWriter, r *http.Request) {
	if !a.requireStore(w) {
		return
	}
	rows, err := a.Store.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]openapi.Project, 0, len(rows))
	for _, p := range rows {
		items = append(items, projectToAPI(p))
	}
	writeJSON(w, http.StatusOK, openapi.ProjectList{Items: items})
}

func (a *API) CreateProject(w http.ResponseWriter, r *http.Request) {
	if !a.requireStore(w) {
		return
	}
	var req openapi.CreateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	if !config.Strategy(req.ConfigStrategy).Valid() {
		badRequest(w, fmt.Errorf("invalid config_strategy %q", req.ConfigStrategy))
		return
	}
	var proj store.Project
	err := a.Store.WithAudit(r.Context(), store.AppendAuditParams{
		Action:  "create_project",
		Actor:   actorFromRequest(r),
		Payload: mustMarshal(map[string]any{"slug": req.Slug}),
	}, func(tx store.Tx) error {
		p, err := tx.CreateProject(r.Context(), req.Slug, req.DisplayName, string(req.ConfigStrategy))
		if err != nil {
			return err
		}
		proj = p
		return nil
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, projectToAPI(proj))
}

func (a *API) GetProject(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projectToAPI(proj))
}

// mustMarshal returns the JSON bytes for v or `{}` if marshalling fails.
// Audit payloads are always small maps the handler constructed; the
// only realistic failure mode is a programming error worth ignoring
// rather than propagating.
func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}
