-- name: CreateEnvironment :one
INSERT INTO environments (id, project_id, slug, name)
VALUES ($1, $2, $3, $4)
RETURNING id, project_id, slug, name, created_at;

-- name: GetEnvironmentBySlug :one
SELECT id, project_id, slug, name, created_at
FROM environments
WHERE project_id = $1 AND slug = $2;

-- name: ListEnvironmentsByProject :many
SELECT id, project_id, slug, name, created_at
FROM environments
WHERE project_id = $1
ORDER BY created_at ASC;
