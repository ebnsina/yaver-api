-- name: UpsertCustomer :one
INSERT INTO customers (id, org_id, phone, name, external_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (org_id, phone) DO UPDATE
SET name        = COALESCE(EXCLUDED.name, customers.name),
    external_id = COALESCE(EXCLUDED.external_id, customers.external_id)
RETURNING id, dnd;

-- name: ListCustomersByOrg :many
SELECT id, phone, name, external_id, dnd, created_at
FROM customers WHERE org_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: SetCustomerDND :exec
UPDATE customers SET dnd = $3 WHERE id = $1 AND org_id = $2;
