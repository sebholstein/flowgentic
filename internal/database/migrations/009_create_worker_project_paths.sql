-- +goose Up
CREATE TABLE worker_project_paths (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    worker_id  TEXT NOT NULL,
    path       TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (project_id, worker_id)
);

-- Migrate existing JSON data.
INSERT INTO worker_project_paths (project_id, worker_id, path)
SELECT p.id, j.key, j.value
FROM projects p, json_each(p.worker_paths) j
WHERE p.worker_paths != '{}' AND p.worker_paths != '';

-- +goose Down
DROP TABLE IF EXISTS worker_project_paths;
