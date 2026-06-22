-- name: NextSnapshotVersion :one
SELECT COALESCE(MAX(version), 0) + 1 AS next_version
FROM snapshots
WHERE project_id = ?;

-- name: CreateSnapshot :one
INSERT INTO snapshots (id, project_id, environment_id, version, compiled)
VALUES (?, ?, ?, ?, ?)
RETURNING id, project_id, environment_id, version, compiled, created_at;

-- name: GetSnapshotByID :one
SELECT id, project_id, environment_id, version, compiled, created_at
FROM snapshots
WHERE project_id = ? AND id = ?;

-- name: GetLatestSnapshot :one
SELECT id, project_id, environment_id, version, compiled, created_at
FROM snapshots
WHERE project_id = ?
ORDER BY version DESC
LIMIT 1;

-- name: ListSnapshotsByProject :many
SELECT id, project_id, environment_id, version, compiled, created_at
FROM snapshots
WHERE project_id = ?
ORDER BY version DESC
LIMIT ?;

-- name: ListLatestFlagVersions :many
SELECT
    flag_key,
    flag_id,
    version,
    strategy,
    compiled,
    published_at
FROM (
    SELECT
        f.key            AS flag_key,
        fv.flag_id       AS flag_id,
        fv.version       AS version,
        fv.strategy      AS strategy,
        fv.compiled      AS compiled,
        fv.published_at  AS published_at,
        ROW_NUMBER() OVER (PARTITION BY fv.flag_id ORDER BY fv.version DESC) AS rn
    FROM flag_versions fv
    INNER JOIN flags f ON f.id = fv.flag_id
    WHERE f.project_id = ?
)
WHERE rn = 1;
