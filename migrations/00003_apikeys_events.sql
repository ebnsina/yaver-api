-- +goose Up
CREATE TABLE api_keys (
    id           TEXT PRIMARY KEY,
    org_id       TEXT NOT NULL,
    prefix       TEXT UNIQUE NOT NULL,   -- indexed lookup prefix (yvr_sk_XXXXXXXX)
    secret_hash  BYTEA NOT NULL,         -- sha256 of the full key
    name         TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);

CREATE TABLE events (
    id                TEXT PRIMARY KEY,
    org_id            TEXT NOT NULL,
    type              TEXT NOT NULL,      -- order_placed | order_cancelled | abandoned_cart
    external_event_id TEXT NOT NULL,      -- merchant-side unique id (idempotency)
    payload           JSONB NOT NULL,
    status            TEXT NOT NULL DEFAULT 'received',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, external_event_id)
);
CREATE INDEX idx_events_org_created ON events (org_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS api_keys;
