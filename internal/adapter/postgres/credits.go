package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
)

type CreditRepo struct {
	pool *pgxpool.Pool
	q    *gen.Queries
}

func NewCreditRepo(pool *pgxpool.Pool) *CreditRepo { return &CreditRepo{pool: pool, q: gen.New(pool)} }

func (r *CreditRepo) Balance(ctx context.Context, orgID domain.OrgID) (int, error) {
	bal, err := r.q.GetBalance(ctx, string(orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return int(bal), nil
}

// TryDeduct subtracts amount only if the balance covers it, recording a ledger
// entry, both in one transaction.
func (r *CreditRepo) TryDeduct(ctx context.Context, orgID domain.OrgID, amount int, reason string) (bool, int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback(ctx)
	q := r.q.WithTx(tx)

	bal, err := q.TryDeductBalance(ctx, gen.TryDeductBalanceParams{OrgID: string(orgID), Balance: int32(amount)})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, nil // insufficient funds — nothing changed
	}
	if err != nil {
		return false, 0, err
	}
	if err := q.AddLedger(ctx, gen.AddLedgerParams{ID: id.New("led"), OrgID: string(orgID), Delta: int32(-amount), Reason: reason}); err != nil {
		return false, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, 0, err
	}
	return true, int(bal), nil
}

func (r *CreditRepo) Grant(ctx context.Context, orgID domain.OrgID, amount int, reason string) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	q := r.q.WithTx(tx)

	if err := q.EnsureCreditAccount(ctx, string(orgID)); err != nil {
		return 0, err
	}
	bal, err := q.AdjustBalance(ctx, gen.AdjustBalanceParams{OrgID: string(orgID), Balance: int32(amount)})
	if err != nil {
		return 0, err
	}
	if err := q.AddLedger(ctx, gen.AddLedgerParams{ID: id.New("led"), OrgID: string(orgID), Delta: int32(amount), Reason: reason}); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return int(bal), nil
}

func (r *CreditRepo) Ledger(ctx context.Context, orgID domain.OrgID, limit int) ([]domain.CreditEntry, error) {
	rows, err := r.q.ListLedger(ctx, gen.ListLedgerParams{OrgID: string(orgID), Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.CreditEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CreditEntry{Delta: int(row.Delta), Reason: row.Reason, CreatedAt: row.CreatedAt})
	}
	return out, nil
}
