package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	dbsqlite "github.com/depot/falseflag/internal/db/sqlite"
)

// CompileSnapshot inserts a new snapshot row, picking the next version
// inside an immediate-mode transaction so concurrent compose-stack
// compiles can't collide on (project_id, version).
func (s *sqliteStore) CompileSnapshot(ctx context.Context, p CompileSnapshotParams) (Snapshot, error) {
	var snap Snapshot
	err := s.withImmediateTx(ctx, func(tx *sql.Tx) error {
		q := s.queries.WithTx(tx)
		next, err := q.NextSnapshotVersion(ctx, sqliteUUIDString(p.ProjectID))
		if err != nil {
			return fmt.Errorf("store: next snapshot version: %w", err)
		}
		row, err := q.CreateSnapshot(ctx, dbsqlite.CreateSnapshotParams{
			ID:            sqliteUUIDString(uuid.New()),
			ProjectID:     sqliteUUIDString(p.ProjectID),
			EnvironmentID: sqliteFromNullUUID(p.EnvironmentID),
			Version:       next,
			Compiled:      sqliteRawMessage(p.Compiled),
		})
		if err != nil {
			return fmt.Errorf("store: create snapshot: %w", err)
		}
		snap = sqliteSnapshotFromRow(row)
		return nil
	})
	if err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

// GetSnapshotByID returns the (project, id)-keyed snapshot.
func (s *sqliteStore) GetSnapshotByID(ctx context.Context, projectID, id uuid.UUID) (Snapshot, error) {
	row, err := s.queries.GetSnapshotByID(ctx, dbsqlite.GetSnapshotByIDParams{
		ProjectID: sqliteUUIDString(projectID),
		ID:        sqliteUUIDString(id),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Snapshot{}, ErrNotFound
		}
		return Snapshot{}, fmt.Errorf("store: get snapshot: %w", err)
	}
	return sqliteSnapshotFromRow(row), nil
}

// GetLatestSnapshot returns the highest-version snapshot.
func (s *sqliteStore) GetLatestSnapshot(ctx context.Context, projectID uuid.UUID) (Snapshot, error) {
	row, err := s.queries.GetLatestSnapshot(ctx, sqliteUUIDString(projectID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Snapshot{}, ErrNotFound
		}
		return Snapshot{}, fmt.Errorf("store: latest snapshot: %w", err)
	}
	return sqliteSnapshotFromRow(row), nil
}

// ListSnapshotsByProject returns snapshots newest-first, up to limit.
func (s *sqliteStore) ListSnapshotsByProject(ctx context.Context, projectID uuid.UUID, limit int32) ([]Snapshot, error) {
	rows, err := s.queries.ListSnapshotsByProject(ctx, dbsqlite.ListSnapshotsByProjectParams{
		ProjectID: sqliteUUIDString(projectID),
		Limit:     int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("store: list snapshots: %w", err)
	}
	out := make([]Snapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, sqliteSnapshotFromRow(r))
	}
	return out, nil
}

// ListLatestFlagVersions returns one row per flag with the latest
// published version's IR.
func (s *sqliteStore) ListLatestFlagVersions(ctx context.Context, projectID uuid.UUID) ([]LatestFlagVersion, error) {
	rows, err := s.queries.ListLatestFlagVersions(ctx, sqliteUUIDString(projectID))
	if err != nil {
		return nil, fmt.Errorf("store: list latest flag versions: %w", err)
	}
	out := make([]LatestFlagVersion, 0, len(rows))
	for _, r := range rows {
		out = append(out, sqliteLatestFlagVersionFromRow(r))
	}
	return out, nil
}
