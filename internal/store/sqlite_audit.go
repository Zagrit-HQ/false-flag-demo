package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	dbsqlite "github.com/depot/falseflag/internal/db/sqlite"
)

// AppendAudit writes a single audit event.
func (s *sqliteStore) AppendAudit(ctx context.Context, p AppendAuditParams) (AuditEvent, error) {
	row, err := s.queries.AppendAuditEvent(ctx, sqliteAuditInsertParams(p))
	if err != nil {
		return AuditEvent{}, fmt.Errorf("store: append audit: %w", err)
	}
	return sqliteAuditFromRow(row), nil
}

func sqliteAuditInsertParams(p AppendAuditParams) dbsqlite.AppendAuditEventParams {
	payload := p.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	return dbsqlite.AppendAuditEventParams{
		ID:        sqliteUUIDString(uuid.New()),
		ProjectID: sqliteFromNullUUID(p.ProjectID),
		FlagID:    sqliteFromNullUUID(p.FlagID),
		Action:    p.Action,
		Actor:     sqliteStringFromString(p.Actor),
		Payload:   string(payload),
	}
}

// ListAuditEvents returns audit events for a project in newest-first
// order, applying filters and cursor.
func (s *sqliteStore) ListAuditEvents(ctx context.Context, p ListAuditEventsParams) ([]AuditEvent, error) {
	projectID := sqliteUUIDString(p.ProjectID)
	rows, err := s.queries.ListAuditEventsByProject(ctx, dbsqlite.ListAuditEventsByProjectParams{
		ProjectID: &projectID,
		Action:    sqliteNullableString(p.Action),
		Actor:     sqliteNullableString(p.Actor),
		From:      sqliteNullableTime(p.From),
		To:        sqliteNullableTime(p.To),
		CursorTs:  sqliteNullableTime(p.CursorTs),
		CursorID:  sqliteNullableCursorID(p),
		Limit:     int64(p.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("store: list audit events: %w", err)
	}
	out := make([]AuditEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, sqliteAuditFromRow(r))
	}
	return out, nil
}

// sqliteNullableString returns nil for empty strings so sqlc's
// narg("IS NULL") branch fires; otherwise the value.
func sqliteNullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func sqliteNullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return sqliteFormatTime(t)
}

func sqliteNullableCursorID(p ListAuditEventsParams) *string {
	if p.CursorTs.IsZero() {
		return nil
	}
	s := p.CursorID.String()
	return &s
}

// withImmediateTx runs fn inside a BEGIN IMMEDIATE transaction and
// retries on SQLITE_BUSY / SQLITE_BUSY_SNAPSHOT. BEGIN IMMEDIATE
// (configured via _txlock=immediate in the DSN) acquires the writer
// lock upfront, which is the closest analog SQLite has to Postgres's
// serializable isolation and the right shape for the
// next-version-then-insert pattern used by snapshot and flag-version
// writes.
func (s *sqliteStore) withImmediateTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	const maxRetries = 8
	for attempt := 0; ; attempt++ {
		err := s.runOnceTx(ctx, fn)
		if err == nil {
			return nil
		}
		if attempt < maxRetries && isSQLiteRetry(err) {
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

func (s *sqliteStore) runOnceTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit: %w", err)
	}
	return nil
}

// sqliteTx is the SQLite implementation of store.Tx. Wraps a
// transaction-scoped *dbsqlite.Queries (created via WithTx on the
// shared *sql.Tx) so callbacks can issue txn-scoped writes.
type sqliteTx struct {
	store *sqliteStore
	q     *dbsqlite.Queries
}

func (t *sqliteTx) PublishFlagVersion(ctx context.Context, p PublishFlagVersionParams) (FlagVersion, error) {
	return t.store.publishFlagVersionTx(ctx, t.q, p)
}

// WithAudit runs fn inside an immediate-mode transaction and appends
// the supplied audit event in the same transaction. Rollback rolls
// the audit row back too. Retries on SQLITE_BUSY / BUSY_SNAPSHOT.
func (s *sqliteStore) WithAudit(ctx context.Context, ev AppendAuditParams, fn func(Tx) error) error {
	return s.withImmediateTx(ctx, func(tx *sql.Tx) error {
		q := s.queries.WithTx(tx)
		if err := fn(&sqliteTx{store: s, q: q}); err != nil {
			return err
		}
		if _, err := q.AppendAuditEvent(ctx, sqliteAuditInsertParams(ev)); err != nil {
			return fmt.Errorf("store: audit append in tx: %w", err)
		}
		return nil
	})
}

// isSQLiteRetry reports whether err carries a SQLITE_BUSY or
// SQLITE_BUSY_SNAPSHOT result code. SQLITE_BUSY_SNAPSHOT (extended
// code 517) is WAL-mode-specific and means "your read snapshot is
// stale; restart the transaction".
func isSQLiteRetry(err error) bool {
	if errors.Is(err, sql.ErrTxDone) {
		return false
	}
	type coded interface{ Code() int }
	var c coded
	if !errors.As(err, &c) {
		return false
	}
	const (
		sqliteBusy         = 5
		sqliteBusySnapshot = 517
	)
	switch c.Code() {
	case sqliteBusy, sqliteBusySnapshot:
		return true
	}
	return false
}
