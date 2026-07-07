package domain

import (
	"context"
	"time"
)

// Customer is a person a merchant may call.
type Customer struct {
	ID         string
	Phone      string
	Name       string
	ExternalID string
	DND        bool // do-not-disturb: never call
	CreatedAt  time.Time
}

// CustomerRepo persists customers and their DND flag.
type CustomerRepo interface {
	// Upsert creates-or-updates a customer by (org, phone), keeping the existing
	// DND flag. Returns the id and current DND.
	Upsert(ctx context.Context, orgID OrgID, phone, name, externalID string) (id string, dnd bool, err error)
	ListByOrg(ctx context.Context, orgID OrgID, limit int) ([]Customer, error)
	SetDND(ctx context.Context, orgID OrgID, id string, dnd bool) error
}
