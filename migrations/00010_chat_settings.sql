-- +goose Up
CREATE TABLE chat_settings (
    org_id       TEXT PRIMARY KEY,
    instructions TEXT NOT NULL DEFAULT '',                         -- system prompt (used by a real LLM)
    widget_title TEXT NOT NULL DEFAULT 'Chat with us',
    welcome      TEXT NOT NULL DEFAULT 'Hi! 👋 How can I help you today?',
    accent       TEXT NOT NULL DEFAULT '#111827',                  -- widget brand color
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS chat_settings;
