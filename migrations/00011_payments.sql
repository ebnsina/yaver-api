-- +goose Up
CREATE TABLE payments (
    id           TEXT PRIMARY KEY,
    org_id       TEXT NOT NULL,
    provider     TEXT NOT NULL,                    -- "mock" | "sslcommerz"
    provider_ref TEXT NOT NULL,                    -- gateway transaction id (tran_id)
    credits      INTEGER NOT NULL,
    amount_bdt   INTEGER NOT NULL,                 -- charged amount in BDT
    status       TEXT NOT NULL DEFAULT 'pending',  -- pending | paid | failed
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    settled_at   TIMESTAMPTZ,
    UNIQUE (provider, provider_ref)
);

CREATE INDEX idx_payments_org ON payments (org_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS payments;
