-- name: CreateSession :exec
INSERT INTO sessions (id, thread_id, worker_id, prompt, status, agent, model, mode, session_mode, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions
WHERE id = ?;

-- name: ListSessionsByThread :many
SELECT * FROM sessions
WHERE thread_id = ?
ORDER BY created_at;

-- name: ListPendingSessions :many
SELECT * FROM sessions
WHERE status = 'pending'
ORDER BY created_at
LIMIT ?;

-- name: UpdateSessionStatus :execresult
UPDATE sessions
SET session_id = ?, status = ?, updated_at = ?
WHERE id = ?;

-- name: GetEmbeddedWorkerPathForSession :one
SELECT p.embedded_worker_path
FROM sessions s
JOIN threads t ON t.id = s.thread_id
JOIN projects p ON p.id = t.project_id
WHERE s.id = ?;

-- name: GetWorkerProjectPath :one
SELECT path FROM worker_project_paths
WHERE project_id = ? AND worker_id = ?;

-- name: InsertSessionEvent :exec
INSERT INTO session_events (session_id, sequence, event_type, payload, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListSessionEventsBySession :many
SELECT * FROM session_events
WHERE session_id = ?
ORDER BY sequence ASC;

-- name: ListSessionEventsByThread :many
SELECT se.* FROM session_events se
JOIN sessions s ON s.id = se.session_id
WHERE s.thread_id = ?
ORDER BY se.sequence ASC;

-- name: ListSessionEventsByTask :many
SELECT se.* FROM session_events se
JOIN sessions s ON s.id = se.session_id
WHERE s.task_id = ?
ORDER BY se.sequence ASC;
