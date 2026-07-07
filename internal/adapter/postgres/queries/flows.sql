-- name: CreateFlow :exec
INSERT INTO flows (id, org_id, name, version, channel, type, locale, spec)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO NOTHING;

-- name: ListFlowsByOrg :many
SELECT id, name, version, channel, type, is_active
FROM flows WHERE org_id = $1 ORDER BY name;

-- name: GetFlowByID :one
SELECT id, org_id, name, version, channel, type, locale, spec
FROM flows WHERE id = $1;

-- name: UpdateFlowSpec :exec
UPDATE flows SET spec = $2 WHERE id = $1 AND org_id = $3;

-- name: GetActiveFlowByName :one
SELECT id, org_id, name, version, channel, type, locale, spec
FROM flows
WHERE org_id = $1 AND name = $2 AND is_active
ORDER BY version DESC
LIMIT 1;
