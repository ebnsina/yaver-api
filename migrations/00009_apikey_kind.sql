-- +goose Up
-- Distinguish secret keys (server-side, yvr_sk_) from publishable keys
-- (embeddable in the browser widget, yvr_pk_).
ALTER TABLE api_keys ADD COLUMN kind TEXT NOT NULL DEFAULT 'secret';

-- +goose Down
ALTER TABLE api_keys DROP COLUMN kind;
