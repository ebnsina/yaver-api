-- +goose Up
CREATE TABLE campaigns (
    id           TEXT PRIMARY KEY,
    org_id       TEXT NOT NULL,
    name         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'draft',   -- draft | completed
    target_count INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at   TIMESTAMPTZ
);
CREATE INDEX idx_campaigns_org ON campaigns (org_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS campaigns;
