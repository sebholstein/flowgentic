-- +goose Up
ALTER TABLE agent_runs RENAME TO sessions;

-- +goose Down
ALTER TABLE sessions RENAME TO agent_runs;
