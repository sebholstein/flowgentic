-- +goose Up
CREATE TABLE session_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    type INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);
CREATE UNIQUE INDEX idx_session_messages_session_seq ON session_messages(session_id, sequence);

-- +goose Down
DROP TABLE session_messages;
