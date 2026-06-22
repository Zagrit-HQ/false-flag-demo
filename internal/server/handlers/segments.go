package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
)

func (a *API) ListSegments(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	rows, err := a.Store.ListSegmentsByProject(r.Context(), proj.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]openapi.Segment, 0, len(rows))
	for _, s := range rows {
		items = append(items, segmentToAPI(s))
	}
	writeJSON(w, http.StatusOK, openapi.SegmentList{Items: items})
}

func (a *API) CreateSegment(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	var req openapi.CreateSegmentRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	predicateRaw, err := normalizePredicate(req.Predicate)
	if err != nil {
		badRequest(w, err)
		return
	}
	var seg store.Segment
	err = a.Store.WithAudit(r.Context(), store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "create_segment",
		Actor:     actorFromRequest(r),
		Payload:   mustMarshal(map[string]any{"key": req.Key}),
	}, func(tx store.Tx) error {
		s, err := tx.CreateSegment(r.Context(), store.CreateSegmentParams{
			ProjectID:   proj.ID,
			Key:         req.Key,
			Name:        req.Name,
			Description: derefString(req.Description),
			Predicate:   predicateRaw,
		})
		if err != nil {
			return err
		}
		seg = s
		return nil
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, segmentToAPI(seg))
}

func (a *API) GetSegment(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, segKey openapi.SegmentKey) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	seg, err := a.Store.GetSegmentByKey(r.Context(), proj.ID, segKey)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, segmentToAPI(seg))
}

func (a *API) UpdateSegment(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, segKey openapi.SegmentKey) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	var req openapi.UpdateSegmentRequest
	if err := decodeJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	predicateRaw, err := normalizePredicate(req.Predicate)
	if err != nil {
		badRequest(w, err)
		return
	}
	var seg store.Segment
	err = a.Store.WithAudit(r.Context(), store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "update_segment",
		Actor:     actorFromRequest(r),
		Payload:   mustMarshal(map[string]any{"key": segKey}),
	}, func(tx store.Tx) error {
		s, err := tx.UpdateSegment(r.Context(), store.UpdateSegmentParams{
			ProjectID:   proj.ID,
			Key:         segKey,
			Name:        req.Name,
			Description: derefString(req.Description),
			Predicate:   predicateRaw,
		})
		if err != nil {
			return err
		}
		seg = s
		return nil
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, segmentToAPI(seg))
}

// normalizePredicate marshals the request's free-form predicate to
// bytes, parses it into config.Predicate, validates it, and returns the
// canonical bytes for storage. CEL predicates are allowed inside
// segments (segments are mutable; the predicate is re-validated on
// every write).
func normalizePredicate(raw any) (json.RawMessage, error) {
	if raw == nil {
		return nil, fmt.Errorf("predicate is required")
	}
	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encoding predicate: %w", err)
	}
	var p config.Predicate
	if err := json.Unmarshal(bytes, &p); err != nil {
		return nil, fmt.Errorf("decoding predicate: %w", err)
	}
	if p.Kind == config.PredSegment {
		// Segments referencing other segments are out of scope for slice 3.
		return nil, fmt.Errorf("segment predicate must not reference another segment")
	}
	if err := config.ValidatePredicate(&p, true); err != nil {
		return nil, err
	}
	return bytes, nil
}
