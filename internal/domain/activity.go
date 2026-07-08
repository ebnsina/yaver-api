package domain

import (
	"context"
	"time"
)

// ActivityEvent is a real-time notification surfaced to the dashboard's live
// feed — a call finishing, a chat message arriving, a campaign starting. It is a
// best-effort UI signal, not a durable record (those go through the outbox).
type ActivityEvent struct {
	Type   string    `json:"type"`             // e.g. "call.completed", "chat.message"
	OrgID  OrgID     `json:"-"`                // routing key; never serialized to clients
	Title  string    `json:"title"`            // short headline
	Detail string    `json:"detail,omitempty"` // optional secondary line
	At     time.Time `json:"at"`
}

// ActivityPublisher publishes live events. Services depend on this narrow port
// so they can announce activity without knowing how it is delivered.
type ActivityPublisher interface {
	PublishActivity(ctx context.Context, e ActivityEvent)
}

// ActivitySubscriber lets a transport (the SSE handler) receive one org's live
// events. The returned unsubscribe func must be called to release the
// subscription; it is safe to call more than once.
type ActivitySubscriber interface {
	SubscribeActivity(orgID OrgID) (<-chan ActivityEvent, func())
}
