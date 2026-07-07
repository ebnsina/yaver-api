-- name: CreateConversation :exec
INSERT INTO conversations (id, org_id) VALUES ($1, $2);

-- name: CreateChannelConversation :exec
INSERT INTO conversations (id, org_id, channel, external_user) VALUES ($1, $2, $3, $4);

-- name: FindOpenChannelConversation :one
SELECT id FROM conversations
WHERE org_id = $1 AND channel = $2 AND external_user = $3 AND status = 'open'
ORDER BY updated_at DESC LIMIT 1;

-- name: GetConversation :one
SELECT id, org_id, channel, status, created_at, updated_at
FROM conversations WHERE id = $1;

-- name: ListConversationsByOrg :many
SELECT c.id, c.status, c.created_at, c.updated_at,
       COALESCE((SELECT content FROM messages m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1), '') AS last_message,
       (SELECT count(*) FROM messages m WHERE m.conversation_id = c.id) AS message_count
FROM conversations c WHERE c.org_id = $1 ORDER BY c.updated_at DESC LIMIT $2;

-- name: TouchConversation :exec
UPDATE conversations SET updated_at = now() WHERE id = $1;

-- name: InsertMessage :exec
INSERT INTO messages (id, conversation_id, role, content) VALUES ($1, $2, $3, $4);

-- name: ListMessages :many
SELECT role, content, created_at FROM messages
WHERE conversation_id = $1 ORDER BY created_at;
