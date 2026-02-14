-- +goose Up
ALTER TABLE agent_runs ADD COLUMN session_mode TEXT NOT NULL DEFAULT 'code';
UPDATE agent_runs SET session_mode = CASE WHEN yolo = 1 THEN 'code' ELSE 'ask' END;
-- SQLite can't DROP COLUMN in older versions; leave yolo as dead column.

-- +goose Down
UPDATE agent_runs SET yolo = CASE WHEN session_mode = 'code' THEN 1 ELSE 0 END;
ALTER TABLE agent_runs DROP COLUMN session_mode;
