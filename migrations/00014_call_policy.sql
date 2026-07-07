-- +goose Up
-- Per-org calling rules: the local-time window outbound calls are allowed in,
-- and the retry count for failed calls. One row per org; absence means defaults.
CREATE TABLE org_call_policy (
    org_id       TEXT PRIMARY KEY,
    window_start SMALLINT NOT NULL DEFAULT 9,  -- local hour, inclusive
    window_end   SMALLINT NOT NULL DEFAULT 21, -- local hour, exclusive
    timezone     TEXT NOT NULL DEFAULT 'Asia/Dhaka',
    max_retries  SMALLINT NOT NULL DEFAULT 2
);

-- +goose Down
DROP TABLE IF EXISTS org_call_policy;
