package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	dbsqlite "github.com/depot/falseflag/internal/db/sqlite"
)

// CreateEnvironment inserts an environment scoped to a project.
func (s *sqliteStore) CreateEnvironment(ctx context.Context, projectID uuid.UUID, slug, name string) (Environment, error) {
	row, err := s.queries.CreateEnvironment(ctx, dbsqlite.CreateEnvironmentParams{
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

// GetEnvironmentBySlug returns the (project, slug)-keyed environment.
func (s *sqliteStore) GetEnvironmentBySlug(ctx context.Context, projectID uuid.UUID, slug string) (Environment, error) {
	row, err := s.queries.GetEnvironmentBySlug(ctx, dbsqlite.GetEnvironmentBySlugParams{
		ProjectID: sqliteUUIDString(projectID),
		Slug:      slug,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Environment{}, ErrNotFound
		}
		return Environment{}, fmt.Errorf("store: get environment: %w", err)
	}
	return sqliteEnvironmentFromRow(row), nil
}

// ListEnvironmentsByProject lists every environment in the project.
func (s *sqliteStore) ListEnvironmentsByProject(ctx context.Context, projectID uuid.UUID) ([]Environment, error) {
	rows, err := s.queries.ListEnvironmentsByProject(ctx, sqliteUUIDString(projectID))
	if err != nil {
		return nil, fmt.Errorf("store: list environments: %w", err)
	}
	out := make([]Environment, 0, len(rows))
	for _, r := range rows {
		out = append(out, sqliteEnvironmentFromRow(r))
	}
	return out, nil
}
