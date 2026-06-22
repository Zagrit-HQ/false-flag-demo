package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	dbsqlite "github.com/depot/falseflag/internal/db/sqlite"
)

// CreateSegment inserts a project-scoped named predicate definition.
func (s *sqliteStore) CreateSegment(ctx context.Context, p CreateSegmentParams) (Segment, error) {
	row, err := s.queries.CreateSegment(ctx, dbsqlite.CreateSegmentParams{
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

// GetSegmentByKey loads the segment by (project, key).
func (s *sqliteStore) GetSegmentByKey(ctx context.Context, projectID uuid.UUID, key string) (Segment, error) {
	row, err := s.queries.GetSegmentByKey(ctx, dbsqlite.GetSegmentByKeyParams{
		ProjectID: sqliteUUIDString(projectID),
		Key:       key,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Segment{}, ErrNotFound
		}
		return Segment{}, fmt.Errorf("store: get segment: %w", err)
	}
	return sqliteSegmentFromRow(row), nil
}

// ListSegmentsByProject lists every segment for the project.
func (s *sqliteStore) ListSegmentsByProject(ctx context.Context, projectID uuid.UUID) ([]Segment, error) {
	rows, err := s.queries.ListSegmentsByProject(ctx, sqliteUUIDString(projectID))
	if err != nil {
		return nil, fmt.Errorf("store: list segments: %w", err)
	}
	out := make([]Segment, 0, len(rows))
	for _, r := range rows {
		out = append(out, sqliteSegmentFromRow(r))
	}
	return out, nil
}

// UpdateSegment overwrites name/description/predicate for an existing
// (project, key) segment.
func (s *sqliteStore) UpdateSegment(ctx context.Context, p UpdateSegmentParams) (Segment, error) {
	row, err := s.queries.UpdateSegment(ctx, dbsqlite.UpdateSegmentParams{
		ProjectID:   sqliteUUIDString(p.ProjectID),
		Key:         p.Key,
		Name:        p.Name,
		Description: p.Description,
		Predicate:   sqliteRawMessage(p.Predicate),
		UpdatedAt:   sqliteFormatTime(time.Now()),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Segment{}, ErrNotFound
		}
		return Segment{}, fmt.Errorf("store: update segment: %w", err)
	}
	return sqliteSegmentFromRow(row), nil
}
