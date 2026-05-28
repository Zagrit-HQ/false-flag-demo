-- name: AppendAuditEvent :one
INSERT INTO audit_events (id, project_id, flag_id, action, actor, payload)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, project_id, flag_id, action, actor, payload, created_at;

-- name: ListAuditEventsByProject :many
SELECT id, project_id, flag_id, action, actor, payload, created_at
FROM audit_events
WHERE project_id = $1
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('actor')::text  IS NULL OR actor  = sqlc.narg('actor')::text)
  AND (sqlc.narg('from')::timestamptz IS NULL OR created_at >= sqlc.narg('from')::timestamptz)
  AND (sqlc.narg('to')::timestamptz   IS NULL OR created_at <  sqlc.narg('to')::timestamptz)
  AND (
        sqlc.narg('cursor_ts')::timestamptz IS NULL
        OR (created_at, id) < (sqlc.narg('cursor_ts')::timestamptz, sqlc.narg('cursor_id')::uuid)
      )
ORDER BY created_at DESC, id DESC
LIMIT $2;
