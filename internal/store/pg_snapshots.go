package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/depot/falseflag/internal/db"
)

// CompileSnapshot inserts a new snapshot row, picking the next version
// for the project under serializable isolation so concurrent compiles
// can't collide on (project_id, version).
func (s *pgStore) CompileSnapshot(ctx context.Context, p CompileSnapshotParams) (Snapshot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Snapshot{}, fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)
	next, err := q.NextSnapshotVersion(ctx, fromUUID(p.ProjectID))
	if err != nil {
		return Snapshot{}, fmt.Errorf("store: next snapshot version: %w", err)
	}
	row, err := q.CreateSnapshot(ctx, db.CreateSnapshotParams{
		ID:            fromUUID(uuid.New()),
		ProjectID:     fromUUID(p.ProjectID),
		EnvironmentID: fromNullUUID(p.EnvironmentID),
		Version:       next,
		Compiled:      p.Compiled,
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("store: create snapshot: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Snapshot{}, fmt.Errorf("store: commit: %w", err)
	}
	return snapshotFromRow(row), nil
}

// GetSnapshotByID returns the (project, id)-keyed snapshot.
func (s *pgStore) GetSnapshotByID(ctx context.Context, projectID, id uuid.UUID) (Snapshot, error) {
	row, err := s.queries.GetSnapshotByID(ctx, db.GetSnapshotByIDParams{
		ProjectID: fromUUID(projectID),
		ID:        fromUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Snapshot{}, ErrNotFound
		}
		return Snapshot{}, fmt.Errorf("store: get snapshot: %w", err)
	}
	return snapshotFromRow(row), nil
}

// GetLatestSnapshot returns the highest-version snapshot for the project.
func (s *pgStore) GetLatestSnapshot(ctx context.Context, projectID uuid.UUID) (Snapshot, error) {
	row, err := s.queries.GetLatestSnapshot(ctx, fromUUID(projectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Snapshot{}, ErrNotFound
		}
		return Snapshot{}, fmt.Errorf("store: latest snapshot: %w", err)
	}
	return snapshotFromRow(row), nil
}

// ListSnapshotsByProject returns snapshots newest-first, up to limit.
func (s *pgStore) ListSnapshotsByProject(ctx context.Context, projectID uuid.UUID, limit int32) ([]Snapshot, error) {
	rows, err := s.queries.ListSnapshotsByProject(ctx, db.ListSnapshotsByProjectParams{
		ProjectID: fromUUID(projectID),
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("store: list snapshots: %w", err)
	}
	out := make([]Snapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, snapshotFromRow(r))
	}
	return out, nil
}

// ListLatestFlagVersions returns one row per flag with the latest
// published version's IR. Used to compile a project snapshot.
func (s *pgStore) ListLatestFlagVersions(ctx context.Context, projectID uuid.UUID) ([]LatestFlagVersion, error) {
	rows, err := s.queries.ListLatestFlagVersions(ctx, fromUUID(projectID))
	if err != nil {
		return nil, fmt.Errorf("store: list latest flag versions: %w", err)
	}
	out := make([]LatestFlagVersion, 0, len(rows))
	for _, r := range rows {
		out = append(out, latestFlagVersionFromRow(r))
	}
	return out, nil
}
