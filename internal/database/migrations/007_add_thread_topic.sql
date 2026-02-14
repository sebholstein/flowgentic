-- +goose Up
ALTER TABLE threads ADD COLUMN topic TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite < 3.35.0 does not support DROP COLUMN; omit for simplicity.
