package store

import (
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// stdlibFromPool returns a *sql.DB that pulls connections from the
// pgxpool. We need this because goose runs on database/sql while the
// rest of the app uses pgx directly. The returned *sql.DB shares the
// pool — closing it would close the pool, so we intentionally don't.
func stdlibFromPool(pool *pgxpool.Pool) *sql.DB {
	return stdlib.OpenDBFromPool(pool)
}
