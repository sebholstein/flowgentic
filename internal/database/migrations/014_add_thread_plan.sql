-- +goose Up
ALTER TABLE threads ADD COLUMN plan TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE threads DROP COLUMN plan;
