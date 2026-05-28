-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS environments (
    id          uuid        PRIMARY KEY,
    project_id  uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    slug        text        NOT NULL,
    name        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, slug)
);

CREATE INDEX IF NOT EXISTS environments_project_id_idx
    ON environments (project_id);

CREATE TABLE IF NOT EXISTS segments (
    id          uuid        PRIMARY KEY,
    project_id  uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key         text        NOT NULL,
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    predicate   jsonb       NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, key)
);

CREATE INDEX IF NOT EXISTS segments_project_id_idx
    ON segments (project_id);

CREATE TABLE IF NOT EXISTS snapshots (
    id              uuid        PRIMARY KEY,
    project_id      uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id  uuid        REFERENCES environments(id) ON DELETE SET NULL,
    version         integer     NOT NULL,
    compiled        jsonb       NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, version)
);

CREATE INDEX IF NOT EXISTS snapshots_project_id_created_at_idx
    ON snapshots (project_id, created_at DESC);

ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS actor text;

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
ALTER TABLE audit_events DROP COLUMN IF EXISTS actor;
DROP INDEX IF EXISTS snapshots_project_id_created_at_idx;
DROP TABLE IF EXISTS snapshots;
DROP INDEX IF EXISTS segments_project_id_idx;
DROP TABLE IF EXISTS segments;
DROP INDEX IF EXISTS environments_project_id_idx;
DROP TABLE IF EXISTS environments;
-- +goose StatementEnd
