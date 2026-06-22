package store

import (
	"fmt"

	"context"

	"github.com/google/uuid"

	dbsqlite "github.com/depot/falseflag/internal/db/sqlite"
)

// sqliteTx methods mirror pgTx; every callback write goes through here
// so it shares the audit-wrapping transaction. The body of each method
// is a direct call into the txn-scoped *dbsqlite.Queries.

func (t *sqliteTx) CreateProject(ctx context.Context, slug, displayName, strategy string) (Project, error) {
	row, err := t.q.CreateProject(ctx, dbsqlite.CreateProjectParams{
		ID:             sqliteUUIDString(uuid.New()),
		Slug:           slug,
		DisplayName:    displayName,
		ConfigStrategy: strategy,
	})
	if err != nil {
		return Project{}, fmt.Errorf("store: create project: %w", err)
	}
	return sqliteProjectFromRow(row), nil
}

func (t *sqliteTx) CreateFlag(ctx context.Context, p CreateFlagParams) (Flag, error) {
	row, err := t.q.CreateFlag(ctx, dbsqlite.CreateFlagParams{
		ID:           sqliteUUIDString(uuid.New()),
		ProjectID:    sqliteUUIDString(p.ProjectID),
		Key:          p.Key,
		Name:         p.Name,
		Description:  p.Description,
		ValueType:    p.ValueType,
		DefaultValue: sqliteRawMessage(p.DefaultValue),
	})
	if err != nil {
		return Flag{}, fmt.Errorf("store: create flag: %w", err)
	}
	return sqliteFlagFromRow(row), nil
}

func (t *sqliteTx) CreateEnvironment(ctx context.Context, projectID uuid.UUID, slug, name string) (Environment, error) {
	row, err := t.q.CreateEnvironment(ctx, dbsqlite.CreateEnvironmentParams{
		ID:        sqliteUUIDString(uuid.New()),
		ProjectID: sqliteUUIDString(projectID),
		Slug:      slug,
		Name:      name,
	})
	if err != nil {
		return Environment{}, fmt.Errorf("store: create environment: %w", err)
	}
	return sqliteEnvironmentFromRow(row), nil
}

func (t *sqliteTx) CreateSegment(ctx context.Context, p CreateSegmentParams) (Segment, error) {
	row, err := t.q.CreateSegment(ctx, dbsqlite.CreateSegmentParams{
		ID:          sqliteUUIDString(uuid.New()),
		ProjectID:   sqliteUUIDString(p.ProjectID),
		Key:         p.Key,
		Name:        p.Name,
		Description: p.Description,
		Predicate:   sqliteRawMessage(p.Predicate),
	})
	if err != nil {
		return Segment{}, fmt.Errorf("store: create segment: %w", err)
	}
	return sqliteSegmentFromRow(row), nil
}

func (t *sqliteTx) UpdateSegment(ctx context.Context, p UpdateSegmentParams) (Segment, error) {
	row, err := t.q.UpdateSegment(ctx, dbsqlite.UpdateSegmentParams{
		ProjectID:   sqliteUUIDString(p.ProjectID),
		Key:         p.Key,
		Name:        p.Name,
		Description: p.Description,
		Predicate:   sqliteRawMessage(p.Predicate),
		UpdatedAt:   sqliteNowString(),
	})
	if err != nil {
		return Segment{}, fmt.Errorf("store: update segment: %w", err)
	}
	return sqliteSegmentFromRow(row), nil
}

// CompileSnapshot inserts a new snapshot inside the open transaction.
// Distinct from sqliteStore.CompileSnapshot, which opens its own tx —
// the Tx version reuses the audit tx so the snapshot and the audit row
// commit together.
func (t *sqliteTx) CompileSnapshot(ctx context.Context, p CompileSnapshotParams) (Snapshot, error) {
	next, err := t.q.NextSnapshotVersion(ctx, sqliteUUIDString(p.ProjectID))
	if err != nil {
		return Snapshot{}, fmt.Errorf("store: next snapshot version: %w", err)
	}
	row, err := t.q.CreateSnapshot(ctx, dbsqlite.CreateSnapshotParams{
		ID:            sqliteUUIDString(uuid.New()),
		ProjectID:     sqliteUUIDString(p.ProjectID),
		EnvironmentID: sqliteFromNullUUID(p.EnvironmentID),
		Version:       next,
		Compiled:      sqliteRawMessage(p.Compiled),
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("store: create snapshot: %w", err)
	}
	return sqliteSnapshotFromRow(row), nil
}
