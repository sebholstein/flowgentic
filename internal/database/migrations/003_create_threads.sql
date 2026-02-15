-- +goose Up
CREATE TABLE threads (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id),
    agent       TEXT NOT NULL,
    model       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_threads_project_id ON threads(project_id);

-- +goose Down
DROP INDEX IF EXISTS idx_threads_project_id;
DROP TABLE IF EXISTS threads;
