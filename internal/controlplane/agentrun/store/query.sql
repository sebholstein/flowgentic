-- name: CreateAgentRun :exec
INSERT INTO agent_runs (id, thread_id, worker_id, prompt, status, agent, model, mode, yolo, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAgentRun :one
SELECT * FROM agent_runs
WHERE id = ?;

-- name: ListAgentRunsByThread :many
SELECT * FROM agent_runs
WHERE thread_id = ?
ORDER BY created_at;

-- name: ListPendingAgentRuns :many
SELECT * FROM agent_runs
WHERE status = 'pending'
ORDER BY created_at
LIMIT ?;

-- name: UpdateAgentRunStatus :execresult
UPDATE agent_runs
SET status = ?, session_id = ?, updated_at = ?
WHERE id = ?;
