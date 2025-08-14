-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
CREATE TABLE videos (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT,
  duration INTEGER DEFAULT 0,
  views INTEGER DEFAULT 0,
  thumbnail_key TEXT,
  visibility TEXT DEFAULT "public",
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  deleted_at INTEGER
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE IF EXISTS videos;
-- +goose StatementEnd
