-- name: CreateFlag :one
INSERT INTO flags (id, project_id, key, name, description, value_type, default_value)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, project_id, key, name, description, value_type, default_value, created_at, updated_at;

-- name: GetFlagByKey :one
SELECT id, project_id, key, name, description, value_type, default_value, created_at, updated_at
FROM flags
WHERE project_id = $1 AND key = $2;

-- name: ListFlagsByProject :many
SELECT id, project_id, key, name, description, value_type, default_value, created_at, updated_at
FROM flags
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: NextFlagVersion :one
SELECT COALESCE(MAX(version), 0) + 1 AS next_version
FROM flag_versions
WHERE flag_id = $1;

-- name: CreateFlagVersion :one
INSERT INTO flag_versions (id, flag_id, version, strategy, source, compiled, source_text)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, flag_id, version, strategy, source, compiled, published_at, source_text;

-- name: GetLatestFlagVersion :one
SELECT id, flag_id, version, strategy, source, compiled, published_at, source_text
FROM flag_versions
WHERE flag_id = $1
ORDER BY version DESC
LIMIT 1;

-- name: ListFlagVersions :many
SELECT id, flag_id, version, strategy, source, compiled, published_at, source_text
FROM flag_versions
WHERE flag_id = $1
ORDER BY version DESC;
