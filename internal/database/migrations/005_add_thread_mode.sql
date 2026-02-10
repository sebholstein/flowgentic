-- +goose Up
ALTER TABLE threads ADD COLUMN mode TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE threads DROP COLUMN mode;
