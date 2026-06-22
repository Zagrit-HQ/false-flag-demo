-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS flags (
    id            TEXT NOT NULL PRIMARY KEY,
    project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key           TEXT NOT NULL,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    value_type    TEXT NOT NULL
        CHECK (value_type IN ('boolean', 'string', 'number', 'object')),
    default_value TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (project_id, key)
);

CREATE INDEX IF NOT EXISTS flags_project_id_idx ON flags (project_id);

CREATE TABLE IF NOT EXISTS flag_versions (
    id           TEXT    NOT NULL PRIMARY KEY,
    flag_id      TEXT    NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    version      INTEGER NOT NULL,
    strategy     TEXT    NOT NULL
        CHECK (strategy IN ('json', 'cel', 'typescript')),
    source       TEXT    NOT NULL,
    compiled     TEXT    NOT NULL,
    published_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (flag_id, version)
);

CREATE INDEX IF NOT EXISTS flag_versions_flag_id_version_idx
    ON flag_versions (flag_id, version DESC);

CREATE TABLE IF NOT EXISTS audit_events (
    id         TEXT NOT NULL PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    flag_id    TEXT REFERENCES flags(id)    ON DELETE SET NULL,
    action     TEXT NOT NULL,
    payload    TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS audit_events_project_id_created_at_idx
    ON audit_events (project_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS flag_versions;
DROP TABLE IF EXISTS flags;
-- +goose StatementEnd
