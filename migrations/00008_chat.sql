-- +goose Up
CREATE TABLE conversations (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL,
    channel     TEXT NOT NULL DEFAULT 'chat',
    customer_id TEXT,
    status      TEXT NOT NULL DEFAULT 'open',  -- open | closed
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_conversations_org ON conversations (org_id, updated_at DESC);

CREATE TABLE messages (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    role            TEXT NOT NULL,  -- user | assistant | system
    content         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_messages_conv ON messages (conversation_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
