-- +goose Up
ALTER TABLE sessions ADD COLUMN task_id TEXT REFERENCES tasks(id) DEFAULT NULL;

-- +goose Down
ALTER TABLE sessions DROP COLUMN task_id;
