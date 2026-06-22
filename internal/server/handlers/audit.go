package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
)

func (a *API) ListAuditEvents(w http.ResponseWriter, r *http.Request, slug openapi.ProjectSlug, params openapi.ListAuditEventsParams) {
	if !a.requireStore(w) {
		return
	}
	proj, err := a.Store.GetProjectBySlug(r.Context(), slug)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	limit := int32(100)
	if params.Limit != nil {
		limit = *params.Limit
	}

	var cursorTs = zeroTime()
	var cursorID uuid.UUID
	if params.Cursor != nil && *params.Cursor != "" {
		ts, id, err := decodeAuditCursor(*params.Cursor)
		if err != nil {
			badRequest(w, fmt.Errorf("invalid cursor: %w", err))
			return
		}
		cursorTs = ts
		cursorID = id
	}

	rows, err := a.Store.ListAuditEvents(r.Context(), store.ListAuditEventsParams{
		ProjectID: proj.ID,
		Action:    derefString(params.Action),
		Actor:     derefString(params.Actor),
		From:      derefTime(params.From),
		To:        derefTime(params.To),
		CursorTs:  cursorTs,
		CursorID:  cursorID,
		// Fetch one extra to detect whether a next page exists.
		Limit: limit + 1,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	items := make([]openapi.AuditEvent, 0, len(rows))
	for i, ev := range rows {
		if int32(i) >= limit {
			break
		}
		items = append(items, auditEventToAPI(ev))
	}

	resp := openapi.AuditEventList{Items: items}
	if int32(len(rows)) > limit {
		last := rows[limit-1]
		cursor := encodeAuditCursor(last.CreatedAt, last.ID)
		resp.NextCursor = &cursor
	}
	writeJSON(w, http.StatusOK, resp)
}

type auditCursor struct {
	Ts string    `json:"ts"`
	ID uuid.UUID `json:"id"`
}

func encodeAuditCursor(ts auditTime, id uuid.UUID) string {
	c := auditCursor{Ts: ts.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), ID: id}
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeAuditCursor(s string) (auditTime, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return zeroTime(), uuid.Nil, err
	}
	var c auditCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return zeroTime(), uuid.Nil, err
	}
	ts, err := parseTime(c.Ts)
	if err != nil {
		return zeroTime(), uuid.Nil, err
	}
	return ts, c.ID, nil
}
