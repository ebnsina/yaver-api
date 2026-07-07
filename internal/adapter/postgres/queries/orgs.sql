-- name: GetOrgByOwner :one
SELECT id, name FROM orgs WHERE owner_user_id = $1;

-- name: CreateOrg :exec
INSERT INTO orgs (id, name, owner_user_id) VALUES ($1, $2, $3)
ON CONFLICT (owner_user_id) DO NOTHING;

-- name: RenameOrg :exec
UPDATE orgs SET name = $2 WHERE id = $1;

-- name: GetOrgOwnerEmail :one
SELECT COALESCE(u.email, '') AS email
FROM orgs o JOIN users u ON u.id = o.owner_user_id
WHERE o.id = $1;
