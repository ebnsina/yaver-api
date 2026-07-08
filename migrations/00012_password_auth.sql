-- +goose Up
-- Email/password sign-up: users may now register without a phone, and carry a
-- bcrypt password hash. Email is unique (case-insensitive) when present.
ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;
ALTER TABLE users ADD COLUMN password_hash BYTEA;
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email)) WHERE email IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS users_email_lower_key;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
-- phone is left nullable on down (re-adding NOT NULL could fail on existing rows).
