-- name: CreateTask :exec
INSERT INTO tasks (id, thread_id, description, subtasks, memory, status, sort_index, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ?;

-- name: ListTasksByThread :many
SELECT * FROM tasks WHERE thread_id = ? ORDER BY sort_index, created_at;

-- name: UpdateTask :execresult
UPDATE tasks
SET description = ?, subtasks = ?, memory = ?, status = ?, sort_index = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;
