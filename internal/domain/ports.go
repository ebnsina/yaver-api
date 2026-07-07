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

// PlaceCallInput is the payload for the place_call job.
type PlaceCallInput struct {
	OrgID   OrgID
	ToPhone string
	FlowID  FlowID
}

// Orchestrator abstracts the durable-job engine. The local dispatcher backs
// Phase 0; the Hatchet adapter (fairness key = merchant_id, retries, cron) slots
// in behind this same interface once the engine is running.
type Orchestrator interface {
	EnqueuePlaceCall(ctx context.Context, in PlaceCallInput) error
}

// CallRepo persists voice interactions.
type CallRepo interface {
	Create(ctx context.Context, c *Call) error
	Get(ctx context.Context, id CallID) (*Call, error)
}

// FlowRepo loads flow definitions.
type FlowRepo interface {
	// GetActiveFlow returns the active flow for (org, name). found=false if none.
	GetActiveFlow(ctx context.Context, orgID OrgID, name string) (f Flow, found bool, err error)
}

// Clock is injected so time is testable.
type Clock interface {
	Now() time.Time
}
