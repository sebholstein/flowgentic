-- +goose Up
ALTER TABLE threads DROP COLUMN agent;
ALTER TABLE threads DROP COLUMN model;

-- +goose Down
ALTER TABLE threads ADD COLUMN agent TEXT NOT NULL DEFAULT '';
ALTER TABLE threads ADD COLUMN model TEXT NOT NULL DEFAULT '';
