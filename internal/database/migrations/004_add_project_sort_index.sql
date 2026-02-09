-- +goose Up
ALTER TABLE projects ADD COLUMN sort_index INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE projects DROP COLUMN sort_index;
