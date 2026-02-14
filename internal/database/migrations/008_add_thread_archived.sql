-- +goose Up
ALTER TABLE threads ADD COLUMN archived INTEGER NOT NULL DEFAULT 0;

-- +goose Down
