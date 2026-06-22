package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/depot/falseflag/internal/db"
)

// pgTx is the Postgres implementation of store.Tx. It wraps a
// transaction-scoped *db.Queries (created via WithTx on the shared
// pgx.Tx) so callbacks can issue txn-scoped writes that commit
// atomically with the surrounding audit append.

func (t *pgTx) CreateProject(ctx context.Context, slug, displayName, strategy string) (Project, error) {
	row, err := t.q.CreateProject(ctx, db.CreateProjectParams{
		ID:             fromUUID(uuid.New()),
		Slug:           slug,
		DisplayName:    displayName,
		ConfigStrategy: strategy,
	})
	if err != nil {
		return Project{}, fmt.Errorf("store: create project: %w", err)
	}
	return projectFromRow(row), nil
}

func (t *pgTx) CreateFlag(ctx context.Context, p CreateFlagParams) (Flag, error) {
	row, err := t.q.CreateFlag(ctx, db.CreateFlagParams{
		ID:           fromUUID(uuid.New()),
		ProjectID:    fromUUID(p.ProjectID),
		Key:          p.Key,
		Name:         p.Name,
		Description:  p.Description,
		ValueType:    p.ValueType,
		DefaultValue: p.DefaultValue,
	})
	if err != nil {
		return Flag{}, fmt.Errorf("store: create flag: %w", err)
	}
	return flagFromRow(row), nil
}

func (t *pgTx) CreateEnvironment(ctx context.Context, projectID uuid.UUID, slug, name string) (Environment, error) {
	row, err := t.q.CreateEnvironment(ctx, db.CreateEnvironmentParams{
		ID:        fromUUID(uuid.New()),
		ProjectID: fromUUID(projectID),
		Slug:      slug,
		Name:      name,
	})
	if err != nil {
		return Environment{}, fmt.Errorf("store: create environment: %w", err)
	}
	return environmentFromRow(row), nil
}

func (t *pgTx) CreateSegment(ctx context.Context, p CreateSegmentParams) (Segment, error) {
	row, err := t.q.CreateSegment(ctx, db.CreateSegmentParams{
		ID:          fromUUID(uuid.New()),
		ProjectID:   fromUUID(p.ProjectID),
		Key:         p.Key,
		Name:        p.Name,
		Description: p.Description,
		Predicate:   p.Predicate,
	})
	if err != nil {
		return Segment{}, fmt.Errorf("store: create segment: %w", err)
	}
	return segmentFromRow(row), nil
}

func (t *pgTx) UpdateSegment(ctx context.Context, p UpdateSegmentParams) (Segment, error) {
	row, err := t.q.UpdateSegment(ctx, db.UpdateSegmentParams{
		ProjectID:   fromUUID(p.ProjectID),
		Key:         p.Key,
		Name:        p.Name,
		Description: p.Description,
		Predicate:   p.Predicate,
		UpdatedAt:   timestampFromTime(time.Now().UTC()),
	})
	if err != nil {
		return Segment{}, fmt.Errorf("store: update segment: %w", err)
	}
	return segmentFromRow(row), nil
}

// CompileSnapshot inserts a new snapshot inside the open transaction.
// Distinct from pgStore.CompileSnapshot, which opens its own tx — the
// Tx version reuses the audit tx so the snapshot and the audit row
// commit together.
func (t *pgTx) CompileSnapshot(ctx context.Context, p CompileSnapshotParams) (Snapshot, error) {
	next, err := t.q.NextSnapshotVersion(ctx, fromUUID(p.ProjectID))
	if err != nil {
		return Snapshot{}, fmt.Errorf("store: next snapshot version: %w", err)
	}
	row, err := t.q.CreateSnapshot(ctx, db.CreateSnapshotParams{
		ID:            fromUUID(uuid.New()),
		ProjectID:     fromUUID(p.ProjectID),
		EnvironmentID: fromNullUUID(p.EnvironmentID),
		Version:       next,
		Compiled:      p.Compiled,
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("store: create snapshot: %w", err)
	}
	return snapshotFromRow(row), nil
}
