package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	dbsqlite "github.com/depot/falseflag/internal/db/sqlite"
)

// CreateProject inserts a project.
func (s *sqliteStore) CreateProject(ctx context.Context, slug, displayName, strategy string) (Project, error) {
	row, err := s.queries.CreateProject(ctx, dbsqlite.CreateProjectParams{
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

// GetProjectBySlug returns the project with the given slug.
func (s *sqliteStore) GetProjectBySlug(ctx context.Context, slug string) (Project, error) {
	row, err := s.queries.GetProjectBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, fmt.Errorf("store: get project: %w", err)
	}
	return sqliteProjectFromRow(row), nil
}

// ListProjects returns all projects, newest first.
func (s *sqliteStore) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.queries.ListProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: list projects: %w", err)
	}
	out := make([]Project, 0, len(rows))
	for _, r := range rows {
		out = append(out, sqliteProjectFromRow(r))
	}
	return out, nil
}
