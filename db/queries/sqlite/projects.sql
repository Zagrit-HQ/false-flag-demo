-- name: ListProjects :many
SELECT id, slug, display_name, config_strategy, created_at, updated_at
FROM projects
ORDER BY created_at DESC;

-- name: GetProjectBySlug :one
SELECT id, slug, display_name, config_strategy, created_at, updated_at
FROM projects
WHERE slug = ?;

-- name: CreateProject :one
INSERT INTO projects (id, slug, display_name, config_strategy)
VALUES (?, ?, ?, ?)
RETURNING id, slug, display_name, config_strategy, created_at, updated_at;
