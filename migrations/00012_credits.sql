-- +goose Up
CREATE TABLE org_credits (
    org_id  TEXT PRIMARY KEY,
    balance INT NOT NULL DEFAULT 0
);

CREATE TABLE credit_ledger (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    delta      INT NOT NULL,   -- +grant / -usage
    reason     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_credit_ledger_org ON credit_ledger (org_id, created_at DESC);

-- Starter grant so existing orgs keep working.
INSERT INTO org_credits (org_id, balance)
SELECT id::text, 500 FROM orgs
ON CONFLICT (org_id) DO NOTHING;
INSERT INTO credit_ledger (id, org_id, delta, reason)
SELECT 'seed_' || id::text, id::text, 500, 'signup_grant' FROM orgs;

-- +goose Down
DROP TABLE IF EXISTS credit_ledger;
DROP TABLE IF EXISTS org_credits;
