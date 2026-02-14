-- +goose Up
ALTER TABLE projects ADD COLUMN agent_planning_task_preferences TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE projects DROP COLUMN agent_planning_task_preferences;
