-- +goose Up
-- One org per owner (Phase 0). Enables get-or-create on first authenticated
-- request; team/multi-user memberships come later.
CREATE UNIQUE INDEX idx_orgs_owner ON orgs (owner_user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_orgs_owner;
