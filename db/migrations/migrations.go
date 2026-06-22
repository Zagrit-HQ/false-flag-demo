// Package migrations bundles the goose migration files via go:embed
// so the API binary can run them on startup without a separate
// `goose up` step. sqlc reads the same .sql files at code-generation
// time; the two consumers cohabit cleanly because sqlc only looks at
// *.sql.
package migrations

import "embed"

// FS holds every numbered goose migration in lexicographic order.
// Postgres migrations live at the top level; SQLite migrations live
// under sqlite/. Callers select a backend via fs.Sub.
//
//go:embed *.sql sqlite/*.sql
var FS embed.FS
