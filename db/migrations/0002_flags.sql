-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS flags (
    id            uuid        PRIMARY KEY,
    project_id    uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key           text        NOT NULL,
    name          text        NOT NULL,
    description   text        NOT NULL DEFAULT '',
    value_type    text        NOT NULL
        CHECK (value_type IN ('boolean', 'string', 'number', 'object')),
    default_value jsonb       NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, key)
);

CREATE INDEX IF NOT EXISTS flags_project_id_idx ON flags (project_id);

CREATE TABLE IF NOT EXISTS flag_versions (
    id           uuid        PRIMARY KEY,
    flag_id      uuid        NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    version      integer     NOT NULL,
    strategy     text        NOT NULL
        CHECK (strategy IN ('json', 'cel', 'typescript')),
    source       jsonb       NOT NULL,
    compiled     jsonb       NOT NULL,
    published_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (flag_id, version)
);

CREATE INDEX IF NOT EXISTS flag_versions_flag_id_version_idx
    ON flag_versions (flag_id, version DESC);

CREATE TABLE IF NOT EXISTS audit_events (
    id         uuid        PRIMARY KEY,
    project_id uuid        REFERENCES projects(id) ON DELETE SET NULL,
    flag_id    uuid        REFERENCES flags(id) ON DELETE SET NULL,
    action     text        NOT NULL,
    payload    jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
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
