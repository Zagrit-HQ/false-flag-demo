-- name: CreateEnvironment :one
INSERT INTO environments (id, project_id, slug, name)
VALUES (?, ?, ?, ?)
RETURNING id, project_id, slug, name, created_at;

-- name: GetEnvironmentBySlug :one
SELECT id, project_id, slug, name, created_at
FROM environments
WHERE project_id = ? AND slug = ?;

-- name: ListEnvironmentsByProject :many
SELECT id, project_id, slug, name, created_at
FROM environments
WHERE project_id = ?
ORDER BY created_at ASC;
