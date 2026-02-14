-- name: GetSecret :one
SELECT secret FROM embedded_worker_config WHERE id = 1;

-- name: UpsertSecret :exec
INSERT INTO embedded_worker_config (id, secret)
VALUES (1, ?)
ON CONFLICT (id) DO UPDATE SET secret = excluded.secret;
