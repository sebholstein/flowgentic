-- +goose Up
CREATE TABLE agent_runs (
    id          TEXT PRIMARY KEY,
    thread_id   TEXT NOT NULL REFERENCES threads(id),
    worker_id   TEXT NOT NULL,
    prompt      TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    agent       TEXT NOT NULL,
    model       TEXT NOT NULL DEFAULT '',
    mode        TEXT NOT NULL DEFAULT '',
    yolo        INTEGER NOT NULL DEFAULT 0,
    session_id  TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_agent_runs_thread_id ON agent_runs(thread_id);
CREATE INDEX idx_agent_runs_status ON agent_runs(status);

-- +goose Down
DROP TABLE agent_runs;
