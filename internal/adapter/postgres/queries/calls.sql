-- name: CreateCall :exec
INSERT INTO calls (id, org_id, flow_id, provider_call_id, direction, status, result)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetCall :one
SELECT id, org_id, flow_id, provider_call_id, direction, status, result, created_at
FROM calls
WHERE id = $1;

-- name: CallSummary :one
SELECT
    count(*)::bigint                                                          AS total,
    count(*) FILTER (WHERE result = 'confirmed')::bigint                      AS confirmed,
    count(*) FILTER (WHERE result = 'cancelled')::bigint                      AS cancelled,
    count(*) FILTER (WHERE created_at >= now() - interval '1 day')::bigint    AS today
FROM calls
WHERE org_id = $1;

-- name: ListCallsByOrg :many
SELECT id, org_id, flow_id, provider_call_id, direction, status, result, created_at
FROM calls
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2;
