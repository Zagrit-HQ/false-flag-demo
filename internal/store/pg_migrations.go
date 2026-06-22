package store

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"

	"github.com/depot/falseflag/db/migrations"
)

// Migrate runs every embedded goose migration against the store's
// pool. Idempotent. Errors propagate so callers can decide whether to
// fail startup or continue degraded.
func (s *pgStore) Migrate(ctx context.Context, log *slog.Logger) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("store: cannot migrate, nil pool")
	}

	sqlDB := stdlibFromPool(s.pool)

	// Postgres migrations live at the top of the embedded FS. SQLite
	// migrations sit under sqlite/. Use the stateful NewProvider API
	// instead of the global goose.SetDialect/SetBaseFS pair so the two
	// backends can cohabit a single test binary without stepping on
	// each other's globals.
	sub, err := fs.Sub(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("store: migrations sub: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, sub)
	if err != nil {
		return fmt.Errorf("store: goose provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("store: goose up: %w", err)
	}
	if log != nil {
		log.Info("migrations applied",
			"backend", BackendPostgres,
			"applied", len(results),
		)
	}
	return nil
}
