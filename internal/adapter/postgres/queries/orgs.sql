-- name: GetOrgByOwner :one
SELECT id, name FROM orgs WHERE owner_user_id = $1;

-- name: CreateOrg :exec
INSERT INTO orgs (id, name, owner_user_id) VALUES ($1, $2, $3)
ON CONFLICT (owner_user_id) DO NOTHING;

-- name: RenameOrg :exec
UPDATE orgs SET name = $2 WHERE id = $1;
