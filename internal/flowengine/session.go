package flowengine

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// Leg is a live voice channel the session runtime drives — one method per
// directive kind. A LiveKit media agent implements it over real audio/DTMF; a
// test implements it with a scripted keypad. Keeping this interface here (pure,
// no telephony imports) is what makes the IVR pipeline testable without a telco.
type Leg interface {
	// Play speaks a prompt (pre-recorded audio or TTS) and returns when done.
	Play(ctx context.Context, p *domain.Prompt) error
	// SayGather plays a prompt then collects a keypress. timedOut is true when no
	// key arrives within the gather's timeout.
	SayGather(ctx context.Context, p *domain.Prompt, g *domain.Gather) (digit string, timedOut bool, err error)
	// Hangup ends the call, optionally after a final prompt.
	Hangup(ctx context.Context, p *domain.Prompt) error
}

// RunSession drives flow f over leg to a terminal outcome. This is the media
// pipeline's control loop: the same logic runs whether the leg is LiveKit audio
// or the mock/test keypad.
func (e *IVR) RunSession(ctx context.Context, f domain.Flow, leg Leg) (*domain.Outcome, error) {
	d, st, out := e.Start(f)
	for {
		switch d.Kind {
		case "hangup":
			if err := leg.Hangup(ctx, d.Prompt); err != nil {
				return nil, err
			}
			return out, nil
		case "say":
			if err := leg.Play(ctx, d.Prompt); err != nil {
				return nil, err
			}
			// A plain say node has no keypad transition; end the call.
			_ = leg.Hangup(ctx, nil)
			return &domain.Outcome{Status: domain.StatusCompleted}, nil
		case "say_gather":
			digit, timedOut, err := leg.SayGather(ctx, d.Prompt, d.Gather)
			if err != nil {
				return nil, err
			}
			if timedOut {
				d, out = e.OnTimeout(f, st)
			} else {
				d, out = e.OnDTMF(f, st, digit)
			}
		default:
			return out, nil
		}
	}
}
