-- +goose Up
-- Pre-release: clear old proto-binary event data. New events are stored as versioned JSON.
DELETE FROM session_events;

-- +goose Down
-- Cannot restore deleted binary data; no-op.
