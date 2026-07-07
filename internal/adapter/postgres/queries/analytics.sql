-- name: ConversationStats :one
-- Per-org conversation totals plus a breakdown by the AI-enriched outcome. Left
-- join so un-summarized conversations still count toward total/today.
SELECT
    count(*)::bigint                                                          AS total,
    count(*) FILTER (WHERE c.created_at >= now() - interval '1 day')::bigint  AS today,
    count(*) FILTER (WHERE i.outcome = 'resolved')::bigint                    AS resolved,
    count(*) FILTER (WHERE i.outcome = 'pending')::bigint                     AS pending,
    count(*) FILTER (WHERE i.outcome = 'sale')::bigint                        AS sale,
    count(*) FILTER (WHERE i.outcome = 'needs_human')::bigint                 AS needs_human
FROM conversations c
LEFT JOIN conversation_insights i ON i.conversation_id = c.id
WHERE c.org_id = $1;

-- name: CreditSpend :one
-- Credits consumed (negative ledger deltas, as a positive number) over two windows.
SELECT
    COALESCE(-sum(delta) FILTER (WHERE delta < 0 AND created_at >= now() - interval '1 day'), 0)::bigint   AS today,
    COALESCE(-sum(delta) FILTER (WHERE delta < 0 AND created_at >= now() - interval '30 days'), 0)::bigint AS last30d
FROM credit_ledger
WHERE org_id = $1;
