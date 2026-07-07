-- +goose Up
CREATE TABLE customers (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL,
    phone       TEXT NOT NULL,          -- E.164
    name        TEXT,
    external_id TEXT,
    dnd         BOOLEAN NOT NULL DEFAULT false,  -- do-not-disturb: skip calls
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, phone)
);
CREATE INDEX idx_customers_org ON customers (org_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS customers;
