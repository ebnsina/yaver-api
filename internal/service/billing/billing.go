// Package billing exposes the org's credit balance, ledger, and top-ups.
package billing

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type Service struct {
	credits domain.CreditRepo
}

func New(credits domain.CreditRepo) *Service { return &Service{credits: credits} }

func (s *Service) Balance(ctx context.Context, orgID domain.OrgID) (int, error) {
	return s.credits.Balance(ctx, orgID)
}

func (s *Service) Ledger(ctx context.Context, orgID domain.OrgID, limit int) ([]domain.CreditEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.credits.Ledger(ctx, orgID, limit)
}

// TopUp adds credits. Real payment capture happens upstream; this records the
// resulting grant.
func (s *Service) TopUp(ctx context.Context, orgID domain.OrgID, amount int) (int, error) {
	if amount <= 0 || amount > 100000 {
		amount = 100
	}
	return s.credits.Grant(ctx, orgID, amount, "topup")
}
