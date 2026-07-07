package domain

import (
	"context"
	"time"
)

// VoiceProvider abstracts telephony. IVR + (later) VA run against this same
// interface; the concrete adapter (mock, livekit) translates directives ↔ wire.
type VoiceProvider interface {
	// PlaceCall dials out and attaches the flow. Returns the provider's call id.
	PlaceCall(ctx context.Context, req CallRequest) (ProviderCallID, error)
}

// ChatTransport is the sibling port for text channels (web widget, WhatsApp,
// Messenger). Defined here so it exists as a seam; implemented from Phase 2.
type ChatTransport interface {
	StartSession(ctx context.Context, orgID OrgID, flow Flow) (string, error)
}

// CallRepo persists voice interactions.
type CallRepo interface {
	Create(ctx context.Context, c *Call) error
	Get(ctx context.Context, id CallID) (*Call, error)
}

// Clock is injected so time is testable.
type Clock interface {
	Now() time.Time
}
