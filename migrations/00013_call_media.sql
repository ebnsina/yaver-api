-- +goose Up
ALTER TABLE calls ADD COLUMN recording_url TEXT;
ALTER TABLE calls ADD COLUMN transcript TEXT;

-- +goose Down
ALTER TABLE calls DROP COLUMN IF EXISTS transcript;
ALTER TABLE calls DROP COLUMN IF EXISTS recording_url;
