package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/depot/falseflag/internal/db"
)

// CreateEnvironment inserts an environment scoped to a project.
func (s *pgStore) CreateEnvironment(ctx context.Context, projectID uuid.UUID, slug, name string) (Environment, error) {
	row, err := s.queries.CreateEnvironment(ctx, db.CreateEnvironmentParams{
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

// GetEnvironmentBySlug returns the (project, slug)-keyed environment.
func (s *pgStore) GetEnvironmentBySlug(ctx context.Context, projectID uuid.UUID, slug string) (Environment, error) {
	row, err := s.queries.GetEnvironmentBySlug(ctx, db.GetEnvironmentBySlugParams{
		ProjectID: fromUUID(projectID),
		Slug:      slug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Environment{}, ErrNotFound
		}
		return Environment{}, fmt.Errorf("store: get environment: %w", err)
	}
	return environmentFromRow(row), nil
}

// ListEnvironmentsByProject lists every environment in the project.
func (s *pgStore) ListEnvironmentsByProject(ctx context.Context, projectID uuid.UUID) ([]Environment, error) {
	rows, err := s.queries.ListEnvironmentsByProject(ctx, fromUUID(projectID))
	if err != nil {
		return nil, fmt.Errorf("store: list environments: %w", err)
	}
	out := make([]Environment, 0, len(rows))
	for _, r := range rows {
		out = append(out, environmentFromRow(r))
	}
	return out, nil
}
