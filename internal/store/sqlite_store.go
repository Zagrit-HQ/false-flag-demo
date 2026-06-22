package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "modernc.org/sqlite" // registers the "sqlite" driver

	dbsqlite "github.com/depot/falseflag/internal/db/sqlite"
)

// sqliteStore is the SQLite-backed implementation of Store. Constructed
// via openSQLite; not exported because callers always go through the
// package-level Open dispatcher.
type sqliteStore struct {
	db      *sql.DB
	queries *dbsqlite.Queries
}

// openSQLite opens a SQLite database at the given driver-native DSN
// (already rewritten to "file:..." by parseBackend) and tunes it for
// the demo workload: WAL journaling, synchronous=NORMAL, foreign keys
// enforced, generous busy_timeout, and BEGIN IMMEDIATE on every
// transaction. MaxOpenConns is pinned to 1 so the single writer never
// contends with itself, which fits the demo-quality scope without
// pretending to be high-concurrency tuning.
func openSQLite(ctx context.Context, dsn string) (*sqliteStore, error) {
	tuned, err := sqliteDSNWithPragmas(dsn)
	if err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite", tuned)
	if err != nil {
		return nil, fmt.Errorf("store: sqlite open: %w", err)
	}
	conn.SetMaxOpenConns(1)
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("store: sqlite ping: %w", err)
	}
	return &sqliteStore{db: conn, queries: dbsqlite.New(conn)}, nil
}

// sqliteDSNWithPragmas augments the operator-supplied DSN with the
// PRAGMAs we always want set. modernc.org/sqlite accepts both
// _pragma=… and _txlock= URI parameters; operators may add their own
// (we don't strip), but if a critical PRAGMA is missing we add it.
func sqliteDSNWithPragmas(dsn string) (string, error) {
	requiredPragmas := []string{
		"journal_mode(WAL)",
		"synchronous(NORMAL)",
		"foreign_keys(ON)",
		"busy_timeout(5000)",
	}
	const txLock = "immediate"

	// Split file path from query string. modernc.org/sqlite uses URI
	// semantics: file:PATH?key=value&key=value.
	pathPart, query, _ := strings.Cut(dsn, "?")
	values, err := url.ParseQuery(query)
	if err != nil {
		return "", fmt.Errorf("store: sqlite dsn: parse query: %w", err)
	}
	existing := values["_pragma"]
	have := make(map[string]struct{}, len(existing))
	for _, p := range existing {
		// Match by leading "name(" so operator-provided variants of
		// the same PRAGMA aren't doubled.
		name, _, _ := strings.Cut(p, "(")
		have[name] = struct{}{}
	}
	for _, p := range requiredPragmas {
		name, _, _ := strings.Cut(p, "(")
		if _, ok := have[name]; !ok {
			values.Add("_pragma", p)
		}
	}
	if values.Get("_txlock") == "" {
		values.Set("_txlock", txLock)
	}
	return pathPart + "?" + values.Encode(), nil
}

// Close releases the underlying *sql.DB.
func (s *sqliteStore) Close() {
	if s == nil || s.db == nil {
		return
	}
	_ = s.db.Close()
}

// Backend reports which storage engine this store is bound to.
func (s *sqliteStore) Backend() Backend { return BackendSQLite }

// TruncateForTest deletes every row from every table the store owns.
// SQLite has no TRUNCATE; per-table DELETE inside one transaction is
// the standard equivalent and runs in microseconds against the test
// dataset.
func (s *sqliteStore) TruncateForTest(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store: truncate: nil store")
	}
	tables := []string{
		"audit_events",
		"snapshots",
		"segments",
		"environments",
		"flag_versions",
		"flags",
		"projects",
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: truncate: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, t := range tables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+t); err != nil {
			return fmt.Errorf("store: truncate %s: %w", t, err)
		}
	}
	return tx.Commit()
}
