// Package logsender is a MessagingSender that logs outbound messages instead of
// calling Meta. Default in dev so the inbound→assistant→outbound flow is
// verifiable without real credentials.
package logsender

import (
	"context"
	"log/slog"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type Sender struct{ log *slog.Logger }

func New(log *slog.Logger) *Sender { return &Sender{log: log} }

func (s *Sender) Send(_ context.Context, m domain.OutboundMessage) error {
	s.log.Info("channel send (log)", "type", m.Type, "to", m.To, "text", m.Text)
	return nil
}
