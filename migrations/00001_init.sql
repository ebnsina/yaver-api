-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Own identity: phone-first users, phone-OTP login, server-side sessions.
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone      TEXT UNIQUE NOT NULL,
    email      TEXT,
    name       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE otp_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       TEXT NOT NULL,
    code_hash   BYTEA NOT NULL,          -- HMAC-SHA256(auth_secret, code)
    attempts    INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_otp_phone_live ON otp_codes (phone) WHERE consumed_at IS NULL;

CREATE TABLE sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash BYTEA UNIQUE NOT NULL,    -- SHA-256 of the opaque cookie token
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_user ON sessions (user_id);

CREATE TABLE orgs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT NOT NULL,
    owner_user_id         UUID NOT NULL REFERENCES users (id),
    credits_balance_paisa BIGINT NOT NULL DEFAULT 0 CHECK (credits_balance_paisa >= 0),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS orgs;
DROP TABLE IF EXISTS users;
