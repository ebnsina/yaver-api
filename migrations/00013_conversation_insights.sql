-- +goose Up
-- AI enrichment for a conversation: a cached summary + outcome the merchant reads
-- in the inbox instead of scrolling the whole transcript. One row per conversation
-- (re-summarizing overwrites), so it's cheap to fetch alongside the messages.
CREATE TABLE conversation_insights (
    conversation_id TEXT PRIMARY KEY REFERENCES conversations (id) ON DELETE CASCADE,
    org_id          TEXT NOT NULL,
    summary         TEXT NOT NULL,
    outcome         TEXT NOT NULL, -- resolved | pending | sale | needs_human | unknown
    sentiment       TEXT NOT NULL, -- positive | neutral | negative
    next_action     TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_conversation_insights_org ON conversation_insights (org_id);

-- +goose Down
DROP TABLE IF EXISTS conversation_insights;
