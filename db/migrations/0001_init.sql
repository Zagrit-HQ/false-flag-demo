-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS projects (
    id           uuid        PRIMARY KEY,
    slug         text        NOT NULL UNIQUE,
    display_name text        NOT NULL,
    config_strategy text     NOT NULL DEFAULT 'json'
        CHECK (config_strategy IN ('json', 'cel', 'typescript')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS projects_created_at_idx ON projects (created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS projects;
-- +goose StatementEnd
