-- +goose Up
CREATE TABLE webhook_endpoints (
    id         TEXT PRIMARY KEY,
    org_id     TEXT UNIQUE NOT NULL,   -- one endpoint per org (Phase 0)
    url        TEXT NOT NULL,
    secret_enc BYTEA NOT NULL,         -- AES-GCM encrypted signing secret
    events     TEXT[] NOT NULL DEFAULT '{}',
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Transactional outbox: written in the same tx as the call outcome.
CREATE TABLE webhook_outbox (
    id            BIGSERIAL PRIMARY KEY,
    org_id        TEXT NOT NULL,
    event         TEXT NOT NULL,
    payload       JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    dispatched_at TIMESTAMPTZ
);
CREATE INDEX idx_outbox_undispatched ON webhook_outbox (id) WHERE dispatched_at IS NULL;

CREATE TABLE webhook_deliveries (
    id               TEXT PRIMARY KEY,
    org_id           TEXT NOT NULL,
    event            TEXT NOT NULL,
    url              TEXT NOT NULL,
    payload          JSONB NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending', -- pending | delivered | dead
    attempts         INT NOT NULL DEFAULT 0,
    last_status_code INT,
    last_error       TEXT,
    next_retry_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_deliveries_due ON webhook_deliveries (next_retry_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhook_outbox;
DROP TABLE IF EXISTS webhook_endpoints;
