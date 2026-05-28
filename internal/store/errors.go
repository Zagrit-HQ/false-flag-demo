package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	sqlitelib "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// Sentinel errors the handlers translate into specific HTTP / Connect
// codes. Wrap them via fmt.Errorf("...: %w", ErrXxx) when adding
// context; callers match with errors.Is.
var (
	// ErrNotFound is returned when a lookup does not match any row.
	ErrNotFound = errors.New("store: not found")

	// ErrConflict is returned when an insert/update violates a uniqueness
	// or foreign-key constraint.
	ErrConflict = errors.New("store: conflict")
)

// IsConflict reports whether err is (or wraps) a uniqueness or
// foreign-key violation from either backend. Handlers use this to map
// store errors to the right HTTP / Connect status without leaking
// driver error codes outward.
func IsConflict(err error) bool {
	if errors.Is(err, ErrConflict) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23505 unique_violation, 23503 foreign_key_violation.
		return pgErr.Code == "23505" || pgErr.Code == "23503"
	}
	var sqliteErr *sqlitelib.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() {
		case sqlite3.SQLITE_CONSTRAINT_UNIQUE,
			sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY,
			sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
			return true
		}
	}
	return false
}
