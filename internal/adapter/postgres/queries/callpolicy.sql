-- name: GetCallPolicy :one
SELECT window_start, window_end, timezone, max_retries
FROM org_call_policy WHERE org_id = $1;

-- name: UpsertCallPolicy :exec
INSERT INTO org_call_policy (org_id, window_start, window_end, timezone, max_retries)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (org_id) DO UPDATE SET
    window_start = excluded.window_start,
    window_end = excluded.window_end,
    timezone = excluded.timezone,
    max_retries = excluded.max_retries;
