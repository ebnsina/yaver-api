-- name: UpsertInsight :exec
INSERT INTO conversation_insights (conversation_id, org_id, summary, outcome, sentiment, next_action)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (conversation_id) DO UPDATE SET
    summary = excluded.summary,
    outcome = excluded.outcome,
    sentiment = excluded.sentiment,
    next_action = excluded.next_action,
    created_at = now();

-- name: GetInsight :one
SELECT summary, outcome, sentiment, next_action, created_at
FROM conversation_insights WHERE conversation_id = $1;
