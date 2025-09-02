-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

ALTER TABLE videos ADD COLUMN status TEXT DEFAULT 'pending_upload';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

ALTER TABLE videos DROP COLUMN status;

-- +goose StatementEnd
