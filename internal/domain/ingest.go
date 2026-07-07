package domain

import (
	"context"
	"time"
)

// APIKeyInfo is public metadata about a key (never the secret).
type APIKeyInfo struct {
	Prefix     string
	Name       string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// IngestEvent is a normalized inbound merchant event ready to persist.
type IngestEvent struct {
	ID              string
	OrgID           string
	Type            string // order_placed | order_cancelled | abandoned_cart
	ExternalEventID string // merchant-side unique id (idempotency)
	Phone           string // E.164, if present
	Payload         []byte // original JSON
}

// APIKeyRepo persists and looks up merchant API keys.
type APIKeyRepo interface {
	Create(ctx context.Context, id, orgID, prefix string, secretHash []byte, name string) error
	// ByPrefix loads a key by its lookup prefix. found=false if absent.
	ByPrefix(ctx context.Context, prefix string) (id, orgID string, secretHash []byte, found bool, err error)
	Touch(ctx context.Context, id string) error
	ListByOrg(ctx context.Context, orgID string) ([]APIKeyInfo, error)
}

// EventRepo stores inbound events idempotently.
type EventRepo interface {
	// Insert stores the event; inserted=false when (org, external_event_id)
	// already exists (duplicate).
	Insert(ctx context.Context, e IngestEvent) (inserted bool, err error)
}
