package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// PaymentRepo persists top-up attempts. Hand-written pgx (not sqlc-generated) so
// the payments table can land without regenerating the typed query layer.
type PaymentRepo struct{ pool *pgxpool.Pool }

func NewPaymentRepo(pool *pgxpool.Pool) *PaymentRepo { return &PaymentRepo{pool: pool} }

func (r *PaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO payments (id, org_id, provider, provider_ref, credits, amount_bdt, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, string(p.OrgID), p.Provider, p.ProviderRef, p.Credits, p.AmountBDT, string(domain.PaymentPending))
	return err
}

// Settle flips a pending payment to the terminal status in one statement. The
// UPDATE only matches rows still pending, so a replayed IPN returns
// transitioned=false and the caller skips the (already-applied) credit grant.
func (r *PaymentRepo) Settle(ctx context.Context, provider, providerRef string, status domain.PaymentStatus) (domain.Payment, bool, error) {
	var p domain.Payment
	var orgID string
	err := r.pool.QueryRow(ctx,
		`UPDATE payments SET status = $3, settled_at = now()
		 WHERE provider = $1 AND provider_ref = $2 AND status = 'pending'
		 RETURNING id, org_id, provider, provider_ref, credits, amount_bdt`,
		provider, providerRef, string(status)).
		Scan(&p.ID, &orgID, &p.Provider, &p.ProviderRef, &p.Credits, &p.AmountBDT)
	if err == nil {
		p.OrgID = domain.OrgID(orgID)
		p.Status = status
		return p, true, nil // this call performed the transition
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Payment{}, false, err
	}

	// Already settled (or unknown ref): load the current row for the caller.
	err = r.pool.QueryRow(ctx,
		`SELECT id, org_id, provider, provider_ref, credits, amount_bdt, status
		 FROM payments WHERE provider = $1 AND provider_ref = $2`,
		provider, providerRef).
		Scan(&p.ID, &orgID, &p.Provider, &p.ProviderRef, &p.Credits, &p.AmountBDT, &p.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Payment{}, false, domain.ErrNotFound
	}
	if err != nil {
		return domain.Payment{}, false, err
	}
	p.OrgID = domain.OrgID(orgID)
	return p, false, nil
}
