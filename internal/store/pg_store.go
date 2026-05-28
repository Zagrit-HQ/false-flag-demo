package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/depot/falseflag/internal/db"
)

// pgStore is the Postgres-backed implementation of Store. Constructed
// via openPostgres; not exported because callers always go through
// the package-level Open dispatcher.
type pgStore struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func openPostgres(ctx context.Context, dsn string) (*pgStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("store: pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &pgStore{
		pool:    pool,
		queries: db.New(pool),
	}, nil
}

// Close releases the pool. Safe to call on a nil store.
func (s *pgStore) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

// Backend reports which storage engine this store is bound to.
func (s *pgStore) Backend() Backend { return BackendPostgres }

// TruncateForTest wipes every table the store owns. Test-only. The
// SQLite analog (sqliteStore.TruncateForTest) runs DELETE FROM per
// table since SQLite has no TRUNCATE.
func (s *pgStore) TruncateForTest(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("store: truncate: nil store")
	}
	_, err := s.pool.Exec(ctx,
		`TRUNCATE TABLE audit_events, snapshots, segments, environments, flag_versions, flags, projects RESTART IDENTITY CASCADE`)
	if err != nil {
		return fmt.Errorf("store: truncate: %w", err)
	}
	return nil
}
