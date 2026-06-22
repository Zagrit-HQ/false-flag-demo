package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/depot/falseflag/internal/db"
)

// CreateProject inserts a project. The caller must validate slug,
// display name, and strategy before calling.
func (s *pgStore) CreateProject(ctx context.Context, slug, displayName, strategy string) (Project, error) {
	row, err := s.queries.CreateProject(ctx, db.CreateProjectParams{
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

// GetProjectBySlug returns the project with the given slug.
func (s *pgStore) GetProjectBySlug(ctx context.Context, slug string) (Project, error) {
	row, err := s.queries.GetProjectBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, fmt.Errorf("store: get project: %w", err)
	}
	return projectFromRow(row), nil
}

// ListProjects returns all projects, newest first.
func (s *pgStore) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.queries.ListProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: list projects: %w", err)
	}
	out := make([]Project, 0, len(rows))
	for _, r := range rows {
		out = append(out, projectFromRow(r))
	}
	return out, nil
}
