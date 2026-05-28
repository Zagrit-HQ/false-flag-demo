package store

import (
	"fmt"
	"strings"
)

// Backend identifies which storage engine an Open call resolved to.
// Today only Postgres is wired; Phase 4 of slice 10 adds SQLite.
type Backend string

const (
	BackendPostgres Backend = "postgres"
	BackendSQLite   Backend = "sqlite"
)

// parseBackend inspects a DSN and returns the resolved backend plus the
// driver-native DSN string. The operator-facing FALSEFLAG_DATABASE_URL
// accepts postgres:// (or postgresql://) and sqlite:// (or file:) schemes;
// the SQLite scheme is rewritten to a database/sql-friendly file: URI.
//
// An empty DSN is rejected explicitly so callers don't accidentally
// pass through a missing env var.
func parseBackend(dsn string) (Backend, string, error) {
	if dsn == "" {
		return "", "", fmt.Errorf("store: empty database URL")
	}
	switch {
	case strings.HasPrefix(dsn, "postgres://"),
		strings.HasPrefix(dsn, "postgresql://"):
		return BackendPostgres, dsn, nil
	case strings.HasPrefix(dsn, "sqlite://"):
		// sqlite:///path → file:/path  (database/sql + modernc.org/sqlite want the file: scheme).
		return BackendSQLite, "file:" + strings.TrimPrefix(dsn, "sqlite://"), nil
	case strings.HasPrefix(dsn, "file:"):
		return BackendSQLite, dsn, nil
	default:
		return "", "", fmt.Errorf("store: unsupported DSN scheme in %q (expected postgres:// or sqlite://)", dsn)
	}
}
