-- name: CreateSegment :one
INSERT INTO segments (id, project_id, key, name, description, predicate)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, project_id, key, name, description, predicate, created_at, updated_at;

-- name: GetSegmentByKey :one
SELECT id, project_id, key, name, description, predicate, created_at, updated_at
FROM segments
WHERE project_id = ? AND key = ?;

-- name: ListSegmentsByProject :many
SELECT id, project_id, key, name, description, predicate, created_at, updated_at
FROM segments
WHERE project_id = ?
ORDER BY created_at ASC;

-- name: UpdateSegment :one
UPDATE segments
SET name        = sqlc.arg('name'),
    description = sqlc.arg('description'),
    predicate   = sqlc.arg('predicate'),
    updated_at  = sqlc.arg('updated_at')
WHERE project_id = sqlc.arg('project_id') AND key = sqlc.arg('key')
RETURNING id, project_id, key, name, description, predicate, created_at, updated_at;
