-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

ALTER TABLE sessions ADD COLUMN expired_at INTEGER NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

ALTER TABLE sessions DROP COLUMN expired_at;

-- +goose StatementEnd
