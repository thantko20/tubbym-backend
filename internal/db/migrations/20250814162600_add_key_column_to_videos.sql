-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
ALTER TABLE videos ADD COLUMN key TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE videos DROP COLUMN key;
-- +goose StatementEnd
