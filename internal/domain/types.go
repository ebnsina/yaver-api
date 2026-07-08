// Package domain holds Yaver's pure business types and port interfaces.
// It imports nothing from adapters, transport, or third-party frameworks —
// dependencies point inward (see CLAUDE.md).
package domain

import "time"

// Identifiers. Kept as distinct string types so they can't be mixed up.
type (
	OrgID  string
	CallID string
	FlowID string
)

// Channel is the transport family a flow runs over.
type Channel string

const (
	ChannelVoice Channel = "voice"
	ChannelChat  Channel = "chat"
)

// FlowType is the runtime kind. Always these names — never the bare word "agent".
type FlowType string

const (
	FlowIVR  FlowType = "ivr"  // deterministic keypad (voice)
	FlowVA   FlowType = "va"   // conversational voice (deferred)
	FlowChat FlowType = "chat" // conversational text
)

// Direction of a voice call.
type Direction string

const (
	Outbound Direction = "outbound"
	Inbound  Direction = "inbound"
)

// CallStatus lifecycle.
type CallStatus string

const (
	StatusQueued     CallStatus = "queued"
	StatusRinging    CallStatus = "ringing"
	StatusInProgress CallStatus = "in_progress"
	StatusCompleted  CallStatus = "completed"
	StatusFailed     CallStatus = "failed"
	StatusNoAnswer   CallStatus = "no_answer"
)

// Flow is the versioned interaction definition. IVR today; VA/chat reuse the
// same envelope with their own spec (see docs/plan.md §5).
type Flow struct {
	ID      FlowID
	OrgID   OrgID
	Name    string
	Version int
	Channel Channel
	Type    FlowType
	Locale  string  // language the agent speaks to the customer (e.g. "en", "bn")
	IVR     IVRSpec // populated when Type == FlowIVR
}

// IVRSpec is a deterministic keypad state graph.
type IVRSpec struct {
	Entry string             `json:"entry"`
	Nodes map[string]IVRNode `json:"nodes"`
}

// IVRNode is one step: play a prompt, optionally gather a keypress, branch,
// or terminate with a result.
type IVRNode struct {
	Say    *Prompt           `json:"say,omitempty"`
	Gather *Gather           `json:"gather,omitempty"`
	On     map[string]string `json:"on,omitempty"` // "1".."9","timeout","invalid" -> next node
	Result string            `json:"result,omitempty"`
	End    bool              `json:"end,omitempty"`
}

// Prompt is static pre-recorded audio or dynamic text rendered via TTS per locale.
type Prompt struct {
	Audio string `json:"audio,omitempty"` // pre-recorded file id
	TTS   string `json:"tts,omitempty"`   // dynamic text; may contain {{slots}}
}

// Gather collects DTMF digits.
type Gather struct {
	Digits   int `json:"digits"`
	TimeoutS int `json:"timeout_s"`
}

// Call is a single voice interaction instance.
type Call struct {
	ID             CallID
	OrgID          OrgID
	FlowID         FlowID
	ProviderCallID ProviderCallID
	Direction      Direction
	Status         CallStatus
	Result         string
	RecordingURL   string // set once the media pipeline records the call
	Transcript     string // set once the recording is transcribed (STT)
	CreatedAt      time.Time
}

// ProviderCallID is the telephony provider's handle for a placed call.
type ProviderCallID string

// CallRequest is what the orchestrator hands a VoiceProvider to place a call.
type CallRequest struct {
	OrgID    OrgID
	ToPhone  string // E.164
	CallerID string // BYON or rented number (Phase 1); empty in Phase 0
	Flow     Flow
}

// Outcome is the terminal, channel-agnostic result of an interaction.
type Outcome struct {
	Result    string
	Status    CallStatus
	DurationS int
}

// LegEvent is a provider-reported event during a live call.
type LegEvent struct {
	Kind  string // "answered" | "dtmf" | "timeout" | "hangup"
	Digit string // set when Kind == "dtmf"
}

// Directive is what the flow engine tells the provider to do next.
type Directive struct {
	Kind   string // "say_gather" | "say" | "hangup"
	Prompt *Prompt
	Gather *Gather
}

// FlowState tracks position within a running flow.
type FlowState struct {
	Node string
}
