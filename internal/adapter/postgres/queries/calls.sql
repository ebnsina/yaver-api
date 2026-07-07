-- name: CreateCall :exec
INSERT INTO calls (id, org_id, flow_id, provider_call_id, direction, status, result)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetCall :one
SELECT id, org_id, flow_id, provider_call_id, direction, status, result, created_at
FROM calls
WHERE id = $1;
