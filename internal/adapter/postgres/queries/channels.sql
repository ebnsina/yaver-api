-- name: UpsertChannel :exec
INSERT INTO channel_connections (id, org_id, type, external_id, access_token, verify_token)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (org_id, type) DO UPDATE
SET external_id = EXCLUDED.external_id,
    access_token = EXCLUDED.access_token,
    verify_token = EXCLUDED.verify_token;

-- name: ListChannelsByOrg :many
SELECT type, external_id, created_at
FROM channel_connections WHERE org_id = $1 ORDER BY type;

-- name: DeleteChannel :exec
DELETE FROM channel_connections WHERE org_id = $1 AND type = $2;

-- name: GetChannelByExternalID :one
SELECT org_id, type, external_id, access_token, verify_token
FROM channel_connections WHERE external_id = $1;

-- name: GetChannelByVerifyToken :one
SELECT org_id, type FROM channel_connections WHERE verify_token = $1 LIMIT 1;
