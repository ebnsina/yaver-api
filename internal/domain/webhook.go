package domain

import (
	"context"
	"time"
)

// OutboxEvent is written in the same transaction as a call outcome. A nil
// OutboxEvent means "persist the call, emit nothing".
type OutboxEvent struct {
	Event   string
	Payload []byte
}

// OutcomeRepo persists a call and (optionally) an outbox row atomically.
type OutcomeRepo interface {
	RecordCallOutcome(ctx context.Context, c *Call, outbox *OutboxEvent) error
}

// WebhookEndpointRow is a merchant's delivery target (secret still encrypted).
type WebhookEndpointRow struct {
	OrgID     string
	URL       string
	SecretEnc []byte
	Events    []string
	Active    bool
}

// DueDelivery is a delivery ready to (re)attempt.
type DueDelivery struct {
	ID       string
	OrgID    string
	Event    string
	URL      string
	Payload  []byte
	Attempts int
}

// WebhookRepo manages endpoints, the outbox drain, and delivery state.
type WebhookRepo interface {
	UpsertEndpoint(ctx context.Context, id, orgID, url string, secretEnc []byte, events []string) error
	GetEndpoint(ctx context.Context, orgID string) (WebhookEndpointRow, bool, error)
	// DrainOutbox claims undispatched outbox rows (SKIP LOCKED), creates a
	// delivery for each org with a subscribed active endpoint, and marks the
	// rows dispatched — all in one transaction. Returns deliveries created.
	DrainOutbox(ctx context.Context, limit int, newDeliveryID func() string) (created int, err error)
	DueDeliveries(ctx context.Context, limit int) ([]DueDelivery, error)
	MarkDelivered(ctx context.Context, id string, statusCode int) error
	Reschedule(ctx context.Context, id string, statusCode int, errMsg, status string, nextRetryAt time.Time) error
}
