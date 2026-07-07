// Package logsender is a domain.EmailSender that logs instead of sending — the
// dev/test default, so the app works without an email provider configured.
package logsender

import (
	"context"
	"log/slog"
)

type Sender struct{ log *slog.Logger }

func New(log *slog.Logger) *Sender { return &Sender{log: log} }

func (s *Sender) Send(_ context.Context, to, subject, body string) error {
	s.log.Info("email (log sender)", "to", to, "subject", subject, "body", body)
	return nil
}
