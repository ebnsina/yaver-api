-- name: UpsertWebhookEndpoint :exec
INSERT INTO webhook_endpoints (id, org_id, url, secret_enc, events)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (org_id) DO UPDATE
SET url = EXCLUDED.url, secret_enc = EXCLUDED.secret_enc, events = EXCLUDED.events, active = true;

-- name: GetWebhookEndpoint :one
SELECT id, org_id, url, secret_enc, events, active
FROM webhook_endpoints WHERE org_id = $1;

-- name: InsertOutbox :exec
INSERT INTO webhook_outbox (org_id, event, payload) VALUES ($1, $2, $3);

-- name: ClaimUndispatchedOutbox :many
SELECT id, org_id, event, payload FROM webhook_outbox
WHERE dispatched_at IS NULL
ORDER BY id
FOR UPDATE SKIP LOCKED
LIMIT $1;

-- name: MarkOutboxDispatched :exec
UPDATE webhook_outbox SET dispatched_at = now() WHERE id = $1;

-- name: CreateDelivery :exec
INSERT INTO webhook_deliveries (id, org_id, event, url, payload) VALUES ($1, $2, $3, $4, $5);

-- name: DueDeliveries :many
SELECT id, org_id, event, url, payload, attempts FROM webhook_deliveries
WHERE status = 'pending' AND next_retry_at <= now()
ORDER BY next_retry_at
LIMIT $1;

-- name: MarkDelivered :exec
UPDATE webhook_deliveries
SET status = 'delivered', attempts = attempts + 1, last_status_code = $2, delivered_at = now()
WHERE id = $1;

-- name: RescheduleDelivery :exec
UPDATE webhook_deliveries
SET status = $4, attempts = attempts + 1, last_status_code = $2, last_error = $3, next_retry_at = $5
WHERE id = $1;
