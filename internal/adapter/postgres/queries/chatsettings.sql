-- name: GetChatSettings :one
SELECT instructions, widget_title, welcome, accent
FROM chat_settings WHERE org_id = $1;

-- name: UpsertChatSettings :exec
INSERT INTO chat_settings (org_id, instructions, widget_title, welcome, accent, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (org_id) DO UPDATE
SET instructions = EXCLUDED.instructions,
    widget_title = EXCLUDED.widget_title,
    welcome      = EXCLUDED.welcome,
    accent       = EXCLUDED.accent,
    updated_at   = now();
