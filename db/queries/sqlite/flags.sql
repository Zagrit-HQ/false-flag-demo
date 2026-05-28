-- name: CreateFlag :one
INSERT INTO flags (id, project_id, key, name, description, value_type, default_value)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, project_id, key, name, description, value_type, default_value, created_at, updated_at;

-- name: GetFlagByKey :one
SELECT id, project_id, key, name, description, value_type, default_value, created_at, updated_at
FROM flags
WHERE project_id = ? AND key = ?;

-- name: ListFlagsByProject :many
SELECT id, project_id, key, name, description, value_type, default_value, created_at, updated_at
FROM flags
WHERE project_id = ?
ORDER BY created_at DESC;

-- name: NextFlagVersion :one
SELECT COALESCE(MAX(version), 0) + 1 AS next_version
FROM flag_versions
WHERE flag_id = ?;

-- name: CreateFlagVersion :one
INSERT INTO flag_versions (id, flag_id, version, strategy, source, compiled, source_text)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, flag_id, version, strategy, source, compiled, published_at, source_text;

-- name: GetLatestFlagVersion :one
SELECT id, flag_id, version, strategy, source, compiled, published_at, source_text
FROM flag_versions
WHERE flag_id = ?
ORDER BY version DESC
LIMIT 1;

-- name: ListFlagVersions :many
SELECT id, flag_id, version, strategy, source, compiled, published_at, source_text
FROM flag_versions
WHERE flag_id = ?
ORDER BY version DESC;
