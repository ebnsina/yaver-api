-- name: CreateCampaign :exec
INSERT INTO campaigns (id, org_id, name) VALUES ($1, $2, $3);

-- name: ListCampaignsByOrg :many
SELECT id, name, status, target_count, created_at, started_at, scheduled_at
FROM campaigns WHERE org_id = $1 ORDER BY created_at DESC;

-- name: GetCampaign :one
SELECT id, org_id, name, status, target_count, created_at, started_at, scheduled_at
FROM campaigns WHERE id = $1;

-- name: StartCampaign :exec
UPDATE campaigns SET status = 'completed', target_count = $3, started_at = now()
WHERE id = $1 AND org_id = $2 AND status IN ('draft', 'scheduled');

-- name: ScheduleCampaign :exec
UPDATE campaigns SET status = 'scheduled', scheduled_at = $3
WHERE id = $1 AND org_id = $2 AND status = 'draft';

-- name: DueCampaigns :many
SELECT id, org_id FROM campaigns
WHERE status = 'scheduled' AND scheduled_at <= $1
ORDER BY scheduled_at;

-- name: AddRecipient :exec
INSERT INTO campaign_recipients (id, campaign_id, org_id, phone, name)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (campaign_id, phone) DO NOTHING;

-- name: ListRecipients :many
SELECT phone, name FROM campaign_recipients WHERE campaign_id = $1 ORDER BY phone;

-- name: CountRecipients :one
SELECT count(*)::bigint FROM campaign_recipients WHERE campaign_id = $1;
