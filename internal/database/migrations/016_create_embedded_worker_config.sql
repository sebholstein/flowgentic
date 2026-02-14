-- +goose Up
CREATE TABLE IF NOT EXISTS embedded_worker_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    secret TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS embedded_worker_config;
