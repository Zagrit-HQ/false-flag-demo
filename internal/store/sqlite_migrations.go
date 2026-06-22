package store

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"

	"github.com/depot/falseflag/db/migrations"
)

// Migrate applies the SQLite migration set (db/migrations/sqlite/) to
// the underlying database. Idempotent. The Postgres backend has a
// sibling Migrate that points at db/migrations/.
func (s *sqliteStore) Migrate(ctx context.Context, log *slog.Logger) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store: cannot migrate, nil db")
	}
	sub, err := fs.Sub(migrations.FS, "sqlite")
	if err != nil {
		return fmt.Errorf("store: sqlite migrations sub: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, s.db, sub)
	if err != nil {
		return fmt.Errorf("store: sqlite goose provider: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("store: sqlite goose up: %w", err)
	}
	if log != nil {
		log.Info("migrations applied",
			"backend", BackendSQLite,
			"applied", len(results),
		)
	}
	return nil
}
