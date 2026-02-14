-- +goose Up
CREATE TABLE tasks (
    id          TEXT PRIMARY KEY,
    thread_id   TEXT NOT NULL REFERENCES threads(id),
    description TEXT NOT NULL DEFAULT '',
    subtasks    TEXT NOT NULL DEFAULT '[]',
    memory      TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    sort_index  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_tasks_thread_id ON tasks(thread_id);

-- +goose Down
DROP TABLE tasks;
