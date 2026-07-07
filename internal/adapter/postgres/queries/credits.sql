-- name: GetBalance :one
SELECT balance FROM org_credits WHERE org_id = $1;

-- name: EnsureCreditAccount :exec
INSERT INTO org_credits (org_id, balance) VALUES ($1, 0) ON CONFLICT (org_id) DO NOTHING;

-- name: AdjustBalance :one
UPDATE org_credits SET balance = balance + $2 WHERE org_id = $1 RETURNING balance;

-- name: TryDeductBalance :one
UPDATE org_credits SET balance = balance - $2
WHERE org_id = $1 AND balance >= $2 RETURNING balance;

-- name: AddLedger :exec
INSERT INTO credit_ledger (id, org_id, delta, reason) VALUES ($1, $2, $3, $4);

-- name: ListLedger :many
SELECT delta, reason, created_at FROM credit_ledger
WHERE org_id = $1 ORDER BY created_at DESC LIMIT $2;
