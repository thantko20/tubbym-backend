-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

ALTER TABLE sessions ADD COLUMN provider TEXT NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

ALTER TABLE sessions DROP COLUMN provider;

-- +goose StatementEnd
