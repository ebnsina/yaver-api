package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

type CallRepo struct {
	q *gen.Queries
}

func NewCallRepo(pool *pgxpool.Pool) *CallRepo { return &CallRepo{q: gen.New(pool)} }

func (r *CallRepo) Create(ctx context.Context, c *domain.Call) error {
	return r.q.CreateCall(ctx, gen.CreateCallParams{
		ID:             string(c.ID),
		OrgID:          string(c.OrgID),
		FlowID:         strPtr(string(c.FlowID)),
		ProviderCallID: strPtr(string(c.ProviderCallID)),
		Direction:      string(c.Direction),
		Status:         string(c.Status),
		Result:         strPtr(c.Result),
	})
}

func (r *CallRepo) Get(ctx context.Context, id domain.CallID) (*domain.Call, error) {
	row, err := r.q.GetCall(ctx, string(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &domain.Call{
		ID:             domain.CallID(row.ID),
		OrgID:          domain.OrgID(row.OrgID),
		FlowID:         domain.FlowID(deref(row.FlowID)),
		ProviderCallID: domain.ProviderCallID(deref(row.ProviderCallID)),
		Direction:      domain.Direction(row.Direction),
		Status:         domain.CallStatus(row.Status),
		Result:         deref(row.Result),
		CreatedAt:      row.CreatedAt,
	}, nil
}

// strPtr maps "" -> NULL so empty optional columns store as NULL.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
