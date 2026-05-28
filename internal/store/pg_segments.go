package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/depot/falseflag/internal/db"
)

// CreateSegment inserts a project-scoped named predicate definition.
// Callers must have validated the predicate JSON before calling.
func (s *pgStore) CreateSegment(ctx context.Context, p CreateSegmentParams) (Segment, error) {
	row, err := s.queries.CreateSegment(ctx, db.CreateSegmentParams{
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

// GetSegmentByKey loads the segment by (project, key).
func (s *pgStore) GetSegmentByKey(ctx context.Context, projectID uuid.UUID, key string) (Segment, error) {
	row, err := s.queries.GetSegmentByKey(ctx, db.GetSegmentByKeyParams{
		ProjectID: fromUUID(projectID),
		Key:       key,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Segment{}, ErrNotFound
		}
		return Segment{}, fmt.Errorf("store: get segment: %w", err)
	}
	return segmentFromRow(row), nil
}

// ListSegmentsByProject lists every segment for the project.
func (s *pgStore) ListSegmentsByProject(ctx context.Context, projectID uuid.UUID) ([]Segment, error) {
	rows, err := s.queries.ListSegmentsByProject(ctx, fromUUID(projectID))
	if err != nil {
		return nil, fmt.Errorf("store: list segments: %w", err)
	}
	out := make([]Segment, 0, len(rows))
	for _, r := range rows {
		out = append(out, segmentFromRow(r))
	}
	return out, nil
}

// UpdateSegment overwrites name/description/predicate for an existing
// (project, key) segment.
func (s *pgStore) UpdateSegment(ctx context.Context, p UpdateSegmentParams) (Segment, error) {
	row, err := s.queries.UpdateSegment(ctx, db.UpdateSegmentParams{
		ProjectID:   fromUUID(p.ProjectID),
		Key:         p.Key,
		Name:        p.Name,
		Description: p.Description,
		Predicate:   p.Predicate,
		UpdatedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Segment{}, ErrNotFound
		}
		return Segment{}, fmt.Errorf("store: update segment: %w", err)
	}
	return segmentFromRow(row), nil
}
