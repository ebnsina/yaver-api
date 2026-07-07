-- name: CreateFlow :exec
INSERT INTO flows (id, org_id, name, version, channel, type, locale, spec)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO NOTHING;

-- name: GetActiveFlowByName :one
SELECT id, org_id, name, version, channel, type, locale, spec
FROM flows
WHERE org_id = $1 AND name = $2 AND is_active
ORDER BY version DESC
LIMIT 1;
