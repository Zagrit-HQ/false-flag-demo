-- +goose Up
-- +goose StatementBegin
ALTER TABLE flag_versions
    ADD COLUMN source_text TEXT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE flag_versions
    DROP COLUMN source_text;
-- +goose StatementEnd
