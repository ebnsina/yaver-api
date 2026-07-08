package domain

import "context"

// SMSSender delivers a text message to a phone (E.164). Provider-agnostic seam:
// a dev "log" sender, a real gateway (Twilio, a BD aggregator) behind the same
// port. Used to deliver login OTPs.
type SMSSender interface {
	Send(ctx context.Context, toPhone, text string) error
}
