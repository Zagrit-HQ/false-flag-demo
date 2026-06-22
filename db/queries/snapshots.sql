-- name: NextSnapshotVersion :one
SELECT COALESCE(MAX(version), 0) + 1 AS next_version
FROM snapshots
WHERE project_id = $1;

-- name: CreateSnapshot :one
INSERT INTO snapshots (id, project_id, environment_id, version, compiled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, project_id, environment_id, version, compiled, created_at;

-- name: GetSnapshotByID :one
SELECT id, project_id, environment_id, version, compiled, created_at
FROM snapshots
WHERE project_id = $1 AND id = $2;

-- name: GetLatestSnapshot :one
SELECT id, project_id, environment_id, version, compiled, created_at
FROM snapshots
WHERE project_id = $1
ORDER BY version DESC
LIMIT 1;

-- name: ListSnapshotsByProject :many
SELECT id, project_id, environment_id, version, compiled, created_at
FROM snapshots
WHERE project_id = $1
ORDER BY version DESC
LIMIT $2;

-- name: ListLatestFlagVersions :many
SELECT DISTINCT ON (fv.flag_id)
    f.key            AS flag_key,
    fv.flag_id       AS flag_id,
    fv.version       AS version,
    fv.strategy      AS strategy,
    fv.compiled      AS compiled,
    fv.published_at  AS published_at
FROM flag_versions fv
INNER JOIN flags f ON f.id = fv.flag_id
WHERE f.project_id = $1
ORDER BY fv.flag_id, fv.version DESC;
