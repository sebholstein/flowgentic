-- name: ListProjects :many
SELECT * FROM projects
ORDER BY sort_index, created_at;

-- name: GetProject :one
SELECT * FROM projects
WHERE id = ?;

-- name: CreateProject :exec
INSERT INTO projects (id, name, default_planner_agent, default_planner_model, embedded_worker_path, worker_paths, created_at, updated_at, sort_index)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateProject :execresult
UPDATE projects
SET name = ?, default_planner_agent = ?, default_planner_model = ?, embedded_worker_path = ?, worker_paths = ?, updated_at = ?, sort_index = ?
WHERE id = ?;

-- name: DeleteProject :execresult
DELETE FROM projects WHERE id = ?;
