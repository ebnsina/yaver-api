// Package logsender is a dev SMSSender that logs the message instead of sending
// it — so the login OTP is visible in the server logs during local development.
// Never select this in production (it would print codes to the log).
package logsender

import (
	"context"
	"log/slog"

	"github.com/ebnsina/yaver-api/internal/domain"
)

var _ domain.SMSSender = (*Sender)(nil)

type Sender struct{ log *slog.Logger }

func New(log *slog.Logger) *Sender { return &Sender{log: log} }

func (s *Sender) Send(_ context.Context, to, text string) error {
	s.log.Info("sms (dev, not sent)", "to", to, "text", text)
	return nil
}
