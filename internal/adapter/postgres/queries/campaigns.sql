-- name: CreateCampaign :exec
INSERT INTO campaigns (id, org_id, name) VALUES ($1, $2, $3);

-- name: ListCampaignsByOrg :many
SELECT id, name, status, target_count, created_at, started_at
FROM campaigns WHERE org_id = $1 ORDER BY created_at DESC;

-- name: GetCampaign :one
SELECT id, org_id, name, status, target_count, created_at, started_at
FROM campaigns WHERE id = $1;

-- name: StartCampaign :exec
UPDATE campaigns SET status = 'completed', target_count = $3, started_at = now()
WHERE id = $1 AND org_id = $2 AND status = 'draft';
