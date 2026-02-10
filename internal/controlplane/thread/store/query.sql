-- name: ListThreads :many
SELECT * FROM threads
WHERE project_id = ?
ORDER BY created_at;

-- name: GetThread :one
SELECT * FROM threads
WHERE id = ?;

-- name: CreateThread :exec
INSERT INTO threads (id, project_id, agent, model, mode, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateThread :execresult
UPDATE threads
SET agent = ?, model = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteThread :execresult
DELETE FROM threads WHERE id = ?;
