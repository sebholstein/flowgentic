-- +goose Up
CREATE TABLE projects (
    id                     TEXT PRIMARY KEY,
    name                   TEXT NOT NULL,
    default_planner_agent  TEXT NOT NULL DEFAULT '',
    default_planner_model  TEXT NOT NULL DEFAULT '',
    embedded_worker_path   TEXT NOT NULL DEFAULT '',
    worker_paths           TEXT NOT NULL DEFAULT '{}',
    created_at             TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at             TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- +goose Down
DROP TABLE IF EXISTS projects;
