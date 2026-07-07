-- +goose Up
CREATE TABLE channel_connections (
    id           TEXT PRIMARY KEY,
    org_id       TEXT NOT NULL,
    type         TEXT NOT NULL,          -- whatsapp | messenger
    external_id  TEXT NOT NULL,          -- WhatsApp phone_number_id or Messenger page_id
    access_token BYTEA NOT NULL,         -- AES-256-GCM encrypted
    verify_token TEXT NOT NULL,          -- Meta webhook verification token
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, type),
    UNIQUE (external_id)
);

-- Chat conversations can now originate from a messaging channel.
ALTER TABLE conversations ADD COLUMN external_user TEXT;

-- +goose Down
ALTER TABLE conversations DROP COLUMN external_user;
DROP TABLE IF EXISTS channel_connections;
