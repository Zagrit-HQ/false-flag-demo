-- name: CreateSegment :one
INSERT INTO segments (id, project_id, key, name, description, predicate)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, project_id, key, name, description, predicate, created_at, updated_at;

-- name: GetSegmentByKey :one
SELECT id, project_id, key, name, description, predicate, created_at, updated_at
FROM segments
WHERE project_id = $1 AND key = $2;

-- name: ListSegmentsByProject :many
SELECT id, project_id, key, name, description, predicate, created_at, updated_at
FROM segments
WHERE project_id = $1
ORDER BY created_at ASC;

-- name: UpdateSegment :one
UPDATE segments
SET name        = $3,
    description = $4,
    predicate   = $5,
    updated_at  = $6
WHERE project_id = $1 AND key = $2
RETURNING id, project_id, key, name, description, predicate, created_at, updated_at;
