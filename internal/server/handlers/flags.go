package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
	"github.com/google/uuid"
)

func (a *API) ListFlags(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	rows, err := a.Store.ListFlagsByProject(r.Context(), proj.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]openapi.Flag, 0, len(rows))
	for _, f := range rows {
		items = append(items, flagToAPI(f))
	}
	writeJSON(w, http.StatusOK, openapi.FlagList{Items: items})
}

func (a *API) CreateFlag(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	var req openapi.CreateFlagRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	defaultRaw, err := json.Marshal(req.DefaultValue)
	if err != nil {
		badRequest(w, fmt.Errorf("encoding default_value: %w", err))
		return
	}
	var flag store.Flag
	err = a.Store.WithAudit(r.Context(), store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "create_flag",
		Actor:     actorFromRequest(r),
		Payload:   mustMarshal(map[string]any{"key": req.Key}),
	}, func(tx store.Tx) error {
		f, err := tx.CreateFlag(r.Context(), store.CreateFlagParams{
			ProjectID:    proj.ID,
			Key:          req.Key,
			Name:         req.Name,
			Description:  derefString(req.Description),
			ValueType:    string(req.ValueType),
			DefaultValue: defaultRaw,
		})
		if err != nil {
			return err
		}
		flag = f
		return nil
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, flagToAPI(flag))
}

func (a *API) GetFlag(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, key openapi.FlagKey) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	flag, err := a.Store.GetFlagByKey(r.Context(), proj.ID, key)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	body := openapi.FlagWithLatest{Flag: flagToAPI(flag)}
	if latest, err := a.Store.GetLatestFlagVersion(r.Context(), flag.ID); err == nil {
		fv := flagVersionToAPI(latest)
		body.LatestVersion = &fv
	} else if !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (a *API) PublishFlagVersion(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, key openapi.FlagKey) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	flag, err := a.Store.GetFlagByKey(r.Context(), proj.ID, key)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	var req openapi.PublishFlagVersionRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	strategy := config.Strategy(req.Strategy)
	if !strategy.Valid() {
		badRequest(w, fmt.Errorf("invalid strategy %q", req.Strategy))
		return
	}
	sourceRaw, err := json.Marshal(req.Source)
	if err != nil {
		badRequest(w, fmt.Errorf("encoding source: %w", err))
		return
	}

	// Choose the compile input. When source_text is supplied and the
	// strategy is typescript, the server treats the raw .ts as the
	// authoritative input — the CLI's pre-compiled IR in `source` is
	// kept for backwards compatibility but is not the source of truth.
	// For json/cel, source_text is stored verbatim for the dashboard's
	// view-source page but does not change the compile input.
	sourceText := derefString(req.SourceText)
	compileInput := sourceRaw
	if strategy == config.StrategyTypeScript && sourceText != "" {
		compileInput = []byte(sourceText)
	}

	// resolveSegmentRefs gracefully no-ops on non-JSON input (TS source),
	// returning the bytes unchanged.
	resolved, err := resolveSegmentRefs(r, a, proj.ID, compileInput)
	if err != nil {
		badRequest(w, err)
		return
	}

	compiled, err := config.Compile(strategy, resolved)
	if err != nil {
		if isCompileError(err) {
			writeCompileError(w, err)
			return
		}
		badRequest(w, err)
		return
	}
	compiledRaw, err := json.Marshal(compiled.IR)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("encoding compiled: %w", err))
		return
	}
	// For TS flags compiled from source_text, keep the persisted
	// `source` jsonb consistent with what the server produced rather
	// than echoing back the CLI's pre-compile.
	persistedSource := sourceRaw
	if strategy == config.StrategyTypeScript && sourceText != "" {
		persistedSource = compiledRaw
	}

	var published store.FlagVersion
	err = a.Store.WithAudit(r.Context(), store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		FlagID:    uuid.NullUUID{UUID: flag.ID, Valid: true},
		Action:    "publish_version",
		Actor:     actorFromRequest(r),
		Payload:   mustMarshal(map[string]any{"strategy": strategy, "flag_key": flag.Key}),
	}, func(tx store.Tx) error {
		v, err := tx.PublishFlagVersion(r.Context(), store.PublishFlagVersionParams{
			FlagID:     flag.ID,
			Strategy:   string(strategy),
			Source:     persistedSource,
			Compiled:   compiledRaw,
			SourceText: sourceText,
		})
		if err != nil {
			return err
		}
		published = v
		return nil
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flagVersionToAPI(published))
}

func (a *API) ListFlagVersions(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, key openapi.FlagKey) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	flag, err := a.Store.GetFlagByKey(r.Context(), proj.ID, key)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	rows, err := a.Store.ListFlagVersions(r.Context(), flag.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]openapi.FlagVersion, 0, len(rows))
	for _, v := range rows {
		items = append(items, flagVersionToAPI(v))
	}
	writeJSON(w, http.StatusOK, openapi.FlagVersionList{Items: items})
}
