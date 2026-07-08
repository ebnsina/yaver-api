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

// CallSummary is a small metrics rollup for the dashboard.
type CallSummary struct {
	Total     int
	Confirmed int
	Cancelled int
	Today     int
}

// CallRepo persists and reads voice interactions.
type CallRepo interface {
	Create(ctx context.Context, c *Call) error
	Get(ctx context.Context, id CallID) (*Call, error)
	ListByOrg(ctx context.Context, orgID OrgID, limit int) ([]Call, error)
	Summary(ctx context.Context, orgID OrgID) (CallSummary, error)
	// AttachMedia stores the recording URL and transcript for a completed call.
	AttachMedia(ctx context.Context, id CallID, recordingURL, transcript string) error
	// DeleteBefore prunes calls older than the cutoff (retention). Returns the
	// number removed.
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// Org is a merchant account.
type Org struct {
	ID   OrgID
	Name string
}

// OrgStore resolves (and lazily creates) a user's org, and renames it. The
// resolve step is the "auto-provision on first authenticated request".
type OrgStore interface {
	EnsureForUser(ctx context.Context, userID, defaultName string) (Org, error)
	Rename(ctx context.Context, orgID OrgID, name string) error
	// OwnerEmail returns the org owner's email; empty string if the user hasn't
	// set one (phone-first signup leaves it blank).
	OwnerEmail(ctx context.Context, orgID OrgID) (string, error)
}

// FlowSummary is list metadata for a flow.
type FlowSummary struct {
	ID      FlowID
	Name    string
	Version int
	Channel Channel
	Type    FlowType
	Active  bool
}

// FlowDetail is a flow with its raw spec JSON (for the builder/editor).
type FlowDetail struct {
	ID      FlowID
	OrgID   OrgID
	Name    string
	Version int
	Channel Channel
	Type    FlowType
	Locale  string
	Spec    []byte // raw JSONB
}

// NewFlow is the input to create a flow (version + id assigned by the repo).
type NewFlow struct {
	Name    string
	Channel Channel
	Type    FlowType
	Locale  string
	Spec    []byte
}

// FlowRepo loads and edits flow definitions.
type FlowRepo interface {
	// GetActiveFlow returns the active flow for (org, name). found=false if none.
	GetActiveFlow(ctx context.Context, orgID OrgID, name string) (f Flow, found bool, err error)
	// Create inserts a new active flow (version 1) and returns its id.
	Create(ctx context.Context, orgID OrgID, nf NewFlow) (FlowID, error)
	ListByOrg(ctx context.Context, orgID OrgID) ([]FlowSummary, error)
	// GetByID returns a flow; found=false if absent. Caller checks org ownership.
	GetByID(ctx context.Context, id FlowID) (fd FlowDetail, found bool, err error)
	// UpdateSpec replaces the spec, scoped to org (no-op if the flow isn't the org's).
	UpdateSpec(ctx context.Context, id FlowID, orgID OrgID, spec []byte) error
}

// Clock is injected so time is testable.
type Clock interface {
	Now() time.Time
}
