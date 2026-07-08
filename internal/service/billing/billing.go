// Package billing exposes the org's credit balance, ledger, and top-ups —
// including real gateway checkout and idempotent IPN settlement.
package billing

import (
	"context"
	"errors"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
)

// bdtPerCredit is the price of one credit in BDT — the pricing lever.
const bdtPerCredit = 1

type Service struct {
	credits  domain.CreditRepo
	payments domain.PaymentRepo
	gateway  domain.PaymentGateway
}

func New(credits domain.CreditRepo, payments domain.PaymentRepo, gateway domain.PaymentGateway) *Service {
	return &Service{credits: credits, payments: payments, gateway: gateway}
}

func (s *Service) Balance(ctx context.Context, orgID domain.OrgID) (int, error) {
	return s.credits.Balance(ctx, orgID)
}

func (s *Service) Ledger(ctx context.Context, orgID domain.OrgID, limit int) ([]domain.CreditEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.credits.Ledger(ctx, orgID, limit)
}

// TopUp grants credits directly, without payment. It backs the dev-only mock
// top-up; the real path is Checkout → gateway → SettlePayment.
func (s *Service) TopUp(ctx context.Context, orgID domain.OrgID, amount int) (int, error) {
	if amount <= 0 || amount > 100000 {
		amount = 100
	}
	return s.credits.Grant(ctx, orgID, amount, "topup")
}

// Checkout opens a hosted payment for `credits` and returns the gateway redirect
// URL. It records a pending payment; credits are granted only later, when the
// gateway confirms the charge via IPN (SettlePayment).
func (s *Service) Checkout(ctx context.Context, orgID domain.OrgID, credits int) (string, error) {
	if credits <= 0 {
		credits = 100
	} else if credits > 100000 {
		credits = 100000
	}
	pid := id.New("pay")
	p := &domain.Payment{
		ID:          pid,
		OrgID:       orgID,
		Provider:    s.gateway.Name(),
		ProviderRef: pid, // we use the payment id as the gateway tran_id
		Credits:     credits,
		AmountBDT:   credits * bdtPerCredit,
		Status:      domain.PaymentPending,
	}
	if err := s.payments.Create(ctx, p); err != nil {
		return "", err
	}
	checkout, err := s.gateway.Checkout(ctx, domain.PaymentRequest{
		OrgID:     orgID,
		PaymentID: pid,
		Credits:   credits,
		AmountBDT: p.AmountBDT,
	})
	if err != nil {
		return "", err
	}
	return checkout.RedirectURL, nil
}

// SettlePayment authenticates a gateway IPN callback and, the first time it
// confirms a successful payment, grants the credits. Safe to call repeatedly:
// the credit grant fires only on the pending→paid transition.
func (s *Service) SettlePayment(ctx context.Context, form map[string]string) error {
	res, ok, err := s.gateway.VerifyIPN(ctx, form)
	if err != nil {
		return err
	}
	if !ok {
		return nil // unauthenticated callback — ignore
	}
	p, transitioned, err := s.payments.Settle(ctx, s.gateway.Name(), res.ProviderRef, res.Status)
	if errors.Is(err, domain.ErrNotFound) {
		return nil // callback for an unknown transaction — ignore
	}
	if err != nil {
		return err
	}
	if transitioned && res.Status == domain.PaymentPaid {
		if _, err := s.credits.Grant(ctx, p.OrgID, p.Credits, "topup"); err != nil {
			return err
		}
	}
	return nil
}
