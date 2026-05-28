package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
	"github.com/google/uuid"
)

func (a *API) ListSnapshots(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, params openapi.ListSnapshotsParams) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	limit := int32(50)
	if params.Limit != nil {
		limit = *params.Limit
	}
	rows, err := a.Store.ListSnapshotsByProject(r.Context(), proj.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]openapi.Snapshot, 0, len(rows))
	for _, s := range rows {
		items = append(items, snapshotToAPI(s))
	}
	writeJSON(w, http.StatusOK, openapi.SnapshotList{Items: items})
}

func (a *API) GetLatestSnapshot(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	snap, err := a.Store.GetLatestSnapshot(r.Context(), proj.ID)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshotToAPI(snap))
}

func (a *API) GetSnapshot(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, id openapi_types.UUID) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	snap, err := a.Store.GetSnapshotByID(r.Context(), proj.ID, uuid.UUID(id))
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshotToAPI(snap))
}

func (a *API) CompileSnapshot(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}

	// Optional body: may select an environment.
	var envID uuid.NullUUID
	if r.ContentLength > 0 {
		var req openapi.CompileSnapshotRequest
		// Body is optional; tolerate empty payload.
		if err := decodeJSON(r, &req); err != nil && !errors.Is(err, errEmptyBody) {
			badRequest(w, err)
			return
		}
		if req.EnvironmentSlug != nil && *req.EnvironmentSlug != "" {
			env, err := a.Store.GetEnvironmentBySlug(r.Context(), proj.ID, *req.EnvironmentSlug)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					badRequest(w, fmt.Errorf("environment %q does not exist", *req.EnvironmentSlug))
					return
				}
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			envID = uuid.NullUUID{UUID: env.ID, Valid: true}
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	bundle, err := a.compileBundle(ctx, proj.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	compiledRaw, err := json.Marshal(bundle)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("encoding bundle: %w", err))
		return
	}

	var snap store.Snapshot
	err = a.Store.WithAudit(ctx, store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "compile_snapshot",
		Actor:     actorFromRequest(r),
		Payload:   mustMarshal(map[string]any{"flag_count": len(bundle.Flags)}),
	}, func(tx store.Tx) error {
		s, err := tx.CompileSnapshot(ctx, store.CompileSnapshotParams{
			ProjectID:     proj.ID,
			EnvironmentID: envID,
			Compiled:      compiledRaw,
		})
		if err != nil {
			return err
		}
		snap = s
		return nil
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, snapshotToAPI(snap))
}

// snapshotBundle is the persisted compiled-bundle shape: a map of
// flag key → rehydrated RulesTree IR. Projects with zero published
// flag versions compile to `{flags: {}}`, not a 5xx.
type snapshotBundle struct {
	Flags map[string]*config.RulesTree `json:"flags"`
}

func (a *API) compileBundle(ctx context.Context, projectID uuid.UUID) (snapshotBundle, error) {
	bundle := snapshotBundle{Flags: map[string]*config.RulesTree{}}
	latest, err := a.Store.ListLatestFlagVersions(ctx, projectID)
	if err != nil {
		return bundle, fmt.Errorf("listing latest versions: %w", err)
	}
	for _, lv := range latest {
		var tree config.RulesTree
		if err := json.Unmarshal(lv.Compiled, &tree); err != nil {
			return bundle, fmt.Errorf("decoding compiled IR for %s: %w", lv.FlagKey, err)
		}
		bundle.Flags[lv.FlagKey] = &tree
	}
	return bundle, nil
}

// errEmptyBody is returned by decodeJSON for the request bodies the
// spec marks as optional. CompileSnapshot tolerates it.
var errEmptyBody = errors.New("empty body")
