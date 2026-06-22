-- name: AppendAuditEvent :one
INSERT INTO audit_events (id, project_id, flag_id, action, actor, payload)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, project_id, flag_id, action, actor, payload, created_at;

-- name: ListAuditEventsByProject :many
SELECT id, project_id, flag_id, action, actor, payload, created_at
FROM audit_events
WHERE project_id = sqlc.arg('project_id')
  AND (sqlc.narg('action') IS NULL OR action = sqlc.narg('action'))
  AND (sqlc.narg('actor')  IS NULL OR actor  = sqlc.narg('actor'))
  AND (sqlc.narg('from')   IS NULL OR created_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')     IS NULL OR created_at <  sqlc.narg('to'))
  AND (
        sqlc.narg('cursor_ts') IS NULL
        OR created_at < sqlc.narg('cursor_ts')
        OR (created_at = sqlc.narg('cursor_ts') AND id < sqlc.narg('cursor_id'))
      )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');
