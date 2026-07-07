-- name: GetActiveFlowByName :one
SELECT id, org_id, name, version, channel, type, locale, spec
FROM flows
WHERE org_id = $1 AND name = $2 AND is_active
ORDER BY version DESC
LIMIT 1;
