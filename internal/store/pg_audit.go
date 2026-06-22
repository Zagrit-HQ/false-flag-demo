package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/depot/falseflag/internal/db"
)

// pgSerializationFailure is Postgres SQLSTATE 40001 — raised by the
// serializable isolation level when concurrent transactions read/write
// overlapping rows. Standard remediation is to retry the whole
// transaction with a fresh snapshot.
const pgSerializationFailure = "40001"

// withAuditMaxRetries caps how many times WithAudit will retry on
// serialization failure. 8 retries × ~10ms backoff covers bursty
// dashboard saves + the parallel edit-corpus e2e suite without
// becoming a runaway loop.
const withAuditMaxRetries = 8

// AppendAudit writes a single audit event. Slice 3 wires this in to
// every mutation handler; persistence failures propagate to the caller
// so the mutation can be rolled back via Store.WithAudit.
func (s *pgStore) AppendAudit(ctx context.Context, p AppendAuditParams) (AuditEvent, error) {
	row, err := s.queries.AppendAuditEvent(ctx, pgAuditInsertParams(p))
	if err != nil {
		return AuditEvent{}, fmt.Errorf("store: append audit: %w", err)
	}
	return auditFromRow(row), nil
}

func pgAuditInsertParams(p AppendAuditParams) db.AppendAuditEventParams {
	payload := p.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	return db.AppendAuditEventParams{
		ID:        fromUUID(uuid.New()),
		ProjectID: fromNullUUID(p.ProjectID),
		FlagID:    fromNullUUID(p.FlagID),
		Action:    p.Action,
		Actor:     textFromString(p.Actor),
		Payload:   payload,
	}
}

// ListAuditEvents returns audit events for a project in newest-first
// order, applying filters and cursor.
func (s *pgStore) ListAuditEvents(ctx context.Context, p ListAuditEventsParams) ([]AuditEvent, error) {
	rows, err := s.queries.ListAuditEventsByProject(ctx, db.ListAuditEventsByProjectParams{
		ProjectID: fromUUID(p.ProjectID),
		Limit:     p.Limit,
		Action:    textFromString(p.Action),
		Actor:     textFromString(p.Actor),
		From:      timestampFromTime(p.From),
		To:        timestampFromTime(p.To),
		CursorTs:  timestampFromTime(p.CursorTs),
		CursorID:  cursorIDParam(p),
	})
	if err != nil {
		return nil, fmt.Errorf("store: list audit events: %w", err)
	}
	out := make([]AuditEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, auditFromRow(r))
	}
	return out, nil
}

// pgTx is the Postgres implementation of store.Tx. It wraps a
// transaction-scoped *db.Queries and exposes only the operations
// callers need inside a WithAudit closure.
type pgTx struct {
	store *pgStore
	q     *db.Queries
}

// PublishFlagVersion publishes a flag version inside the open
// transaction. The next-version read and the insert share the same
// snapshot, so concurrent compose-stack writes can't reuse a version.
func (t *pgTx) PublishFlagVersion(ctx context.Context, p PublishFlagVersionParams) (FlagVersion, error) {
	return t.store.publishFlagVersionTx(ctx, t.q, p)
}

// WithAudit runs fn inside a serializable transaction and, on success,
// appends the supplied audit event in the same transaction. Mutation
// rollback rolls back the audit row too. Serializable isolation matches
// what PublishFlagVersionStandalone uses so the (flag_id, version)
// monotonic-write contract holds.
//
// On Postgres serialization failures (SQLSTATE 40001) the whole
// transaction is retried with a fresh snapshot. fn must therefore be
// idempotent across retries — which is naturally the case for the
// PublishFlagVersion callers that read the latest version, compute
// the next, and insert, all within the txn.
func (s *pgStore) WithAudit(ctx context.Context, ev AppendAuditParams, fn func(Tx) error) error {
	for attempt := 0; ; attempt++ {
		err := s.withAuditOnce(ctx, ev, fn)
		if err == nil {
			return nil
		}
		if attempt < withAuditMaxRetries && isSerializationFailure(err) {
			// Brief jittered backoff before retry: 5ms + attempt*5ms.
			// Keeps wall-clock low while letting the conflicting
			// transaction commit and free up its row locks.
			t := time.NewTimer(time.Duration(5+attempt*5) * time.Millisecond)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
			continue
		}
		return err
	}
}

func (s *pgStore) withAuditOnce(ctx context.Context, ev AppendAuditParams, fn func(Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)
	if err := fn(&pgTx{store: s, q: q}); err != nil {
		return err
	}
	if _, err := q.AppendAuditEvent(ctx, pgAuditInsertParams(ev)); err != nil {
		return fmt.Errorf("store: audit append in tx: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit: %w", err)
	}
	return nil
}

// isSerializationFailure reports whether err carries a Postgres
// SQLSTATE 40001 (serialization_failure) anywhere in its chain.
func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgSerializationFailure {
		return true
	}
	return false
}

func cursorIDParam(p ListAuditEventsParams) pgtype.UUID {
	if p.CursorTs.IsZero() {
		return pgtype.UUID{}
	}
	return fromUUID(p.CursorID)
}

func textFromString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func textToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func timestampFromTime(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}
