// Package notify sends merchant-facing account notifications over email. It
// implements domain.CreditNotifier (fire-and-forget) so hot paths can call it
// without handling delivery errors.
package notify

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type Service struct {
	log   *slog.Logger
	email domain.EmailSender
	orgs  domain.OrgStore
}

func New(log *slog.Logger, email domain.EmailSender, orgs domain.OrgStore) *Service {
	return &Service{log: log, email: email, orgs: orgs}
}

// LowBalance emails the org owner that their credits are running low. Best-
// effort: silently skips when the owner has no email, and only logs on failure
// so a callable path never breaks because of a notification.
func (s *Service) LowBalance(ctx context.Context, orgID domain.OrgID, balance int) {
	to, err := s.orgs.OwnerEmail(ctx, orgID)
	if err != nil {
		s.log.Warn("notify: resolve owner email", "org", orgID, "err", err)
		return
	}
	if to == "" {
		return // phone-first signup, no email on file
	}
	subject := "Your Yaver credits are running low"
	body := fmt.Sprintf(
		"Your Yaver balance is down to %d credits. Top up to keep your calls and chats running without interruption.",
		balance,
	)
	if err := s.email.Send(ctx, to, subject, body); err != nil {
		s.log.Error("notify: send low-balance email", "org", orgID, "err", err)
	}
}
