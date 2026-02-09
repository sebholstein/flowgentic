-- name: ListWorkers :many
SELECT * FROM workers
ORDER BY created_at;

-- name: GetWorker :one
SELECT * FROM workers
WHERE id = ?;

-- name: CreateWorker :exec
INSERT INTO workers (id, name, url, secret, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateWorker :execresult
UPDATE workers
SET name = ?, url = ?, secret = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteWorker :execresult
DELETE FROM workers WHERE id = ?;
