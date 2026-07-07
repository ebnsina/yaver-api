-- +goose Up
-- Campaigns can carry an explicit recipient list (CSV import) instead of blasting
-- every callable customer, and can be scheduled to auto-start at a future time.
ALTER TABLE campaigns ADD COLUMN scheduled_at TIMESTAMPTZ; -- set when status = 'scheduled'

CREATE TABLE campaign_recipients (
    id          TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE,
    org_id      TEXT NOT NULL,
    phone       TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    UNIQUE (campaign_id, phone)  -- dedup within a campaign
);
CREATE INDEX idx_campaign_recipients_campaign ON campaign_recipients (campaign_id);

-- +goose Down
DROP TABLE IF EXISTS campaign_recipients;
ALTER TABLE campaigns DROP COLUMN IF EXISTS scheduled_at;
