package domain

import (
	"context"
	"time"
)

// CallPolicy is per-org calling rules: the local-time window during which
// outbound calls are allowed, and how many times to retry a failed call. Keeps
// Yaver from dialing customers in the middle of the night.
type CallPolicy struct {
	WindowStart int    // local hour calls may start, inclusive [0..23]
	WindowEnd   int    // local hour calls must stop, exclusive [1..24]
	Timezone    string // IANA name, e.g. "Asia/Dhaka"
	MaxRetries  int    // retry attempts for a failed/no-answer call
}

// DefaultCallPolicy is a sane BD default: 9am–9pm Dhaka time, retry twice.
func DefaultCallPolicy() CallPolicy {
	return CallPolicy{WindowStart: 9, WindowEnd: 21, Timezone: "Asia/Dhaka", MaxRetries: 2}
}

// Allows reports whether a call may be placed at instant t under this policy.
// The window is evaluated in the policy's timezone; an unknown timezone falls
// back to UTC so a bad config never silently blocks every call.
func (p CallPolicy) Allows(t time.Time) bool {
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		loc = time.UTC
	}
	h := t.In(loc).Hour()
	return h >= p.WindowStart && h < p.WindowEnd
}

// Valid reports whether the policy's fields are in range and the window is a
// non-empty daytime span (no wrap-around).
func (p CallPolicy) Valid() bool {
	if p.WindowStart < 0 || p.WindowStart > 23 || p.WindowEnd < 1 || p.WindowEnd > 24 {
		return false
	}
	if p.WindowStart >= p.WindowEnd {
		return false
	}
	if p.MaxRetries < 0 || p.MaxRetries > 10 {
		return false
	}
	if _, err := time.LoadLocation(p.Timezone); err != nil {
		return false
	}
	return true
}

// CallPolicyRepo persists the per-org calling policy.
type CallPolicyRepo interface {
	// Get returns the org's policy, or DefaultCallPolicy if none is saved.
	Get(ctx context.Context, orgID OrgID) (CallPolicy, error)
	Upsert(ctx context.Context, orgID OrgID, p CallPolicy) error
}
