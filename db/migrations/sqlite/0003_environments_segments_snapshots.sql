-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS environments (
    id          TEXT NOT NULL PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    slug        TEXT NOT NULL,
    name        TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (project_id, slug)
);

CREATE INDEX IF NOT EXISTS environments_project_id_idx
    ON environments (project_id);

CREATE TABLE IF NOT EXISTS segments (
    id          TEXT NOT NULL PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    predicate   TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (project_id, key)
);

CREATE INDEX IF NOT EXISTS segments_project_id_idx
    ON segments (project_id);

CREATE TABLE IF NOT EXISTS snapshots (
    id              TEXT    NOT NULL PRIMARY KEY,
    project_id      TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id  TEXT    REFERENCES environments(id) ON DELETE SET NULL,
    version         INTEGER NOT NULL,
    compiled        TEXT    NOT NULL,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (project_id, version)
);

CREATE INDEX IF NOT EXISTS snapshots_project_id_created_at_idx
    ON snapshots (project_id, created_at DESC);

ALTER TABLE audit_events ADD COLUMN actor TEXT;

CREATE INDEX IF NOT EXISTS audit_events_action_idx
    ON audit_events (action);

CREATE INDEX IF NOT EXISTS audit_events_actor_idx
    ON audit_events (actor)
    WHERE actor IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS audit_events_actor_idx;
DROP INDEX IF EXISTS audit_events_action_idx;
ALTER TABLE audit_events DROP COLUMN actor;
DROP INDEX IF EXISTS snapshots_project_id_created_at_idx;
DROP TABLE IF EXISTS snapshots;
DROP INDEX IF EXISTS segments_project_id_idx;
DROP TABLE IF EXISTS segments;
DROP INDEX IF EXISTS environments_project_id_idx;
DROP TABLE IF EXISTS environments;
-- +goose StatementEnd
