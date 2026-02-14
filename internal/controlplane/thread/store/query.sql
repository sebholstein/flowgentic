-- name: ListThreads :many
SELECT * FROM threads
WHERE project_id = ?
ORDER BY created_at DESC;

-- name: GetThread :one
SELECT * FROM threads
WHERE id = ?;

-- name: CreateThread :exec
INSERT INTO threads (id, project_id, mode, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateThreadTopic :execresult
UPDATE threads
SET topic = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateThreadArchived :execresult
UPDATE threads
SET archived = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateThreadPlan :execresult
UPDATE threads
SET plan = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteThread :execresult
DELETE FROM threads WHERE id = ?;
