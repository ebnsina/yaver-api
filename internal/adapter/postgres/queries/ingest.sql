-- name: CreateAPIKey :exec
INSERT INTO api_keys (id, org_id, prefix, secret_hash, name)
VALUES ($1, $2, $3, $4, $5);

-- name: ListAPIKeysByOrg :many
SELECT prefix, name, created_at, last_used_at
FROM api_keys WHERE org_id = $1 ORDER BY created_at DESC;

-- name: GetAPIKeyByPrefix :one
SELECT id, org_id, secret_hash FROM api_keys WHERE prefix = $1;

-- name: TouchAPIKey :exec
UPDATE api_keys SET last_used_at = now() WHERE id = $1;

-- name: InsertEvent :one
INSERT INTO events (id, org_id, type, external_event_id, payload)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (org_id, external_event_id) DO NOTHING
RETURNING id;
