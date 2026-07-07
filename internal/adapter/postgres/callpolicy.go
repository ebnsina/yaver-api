package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

type CallPolicyRepo struct{ q *gen.Queries }

func NewCallPolicyRepo(pool *pgxpool.Pool) *CallPolicyRepo { return &CallPolicyRepo{q: gen.New(pool)} }

func (r *CallPolicyRepo) Get(ctx context.Context, orgID domain.OrgID) (domain.CallPolicy, error) {
	row, err := r.q.GetCallPolicy(ctx, string(orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DefaultCallPolicy(), nil
	}
	if err != nil {
		return domain.CallPolicy{}, err
	}
	return domain.CallPolicy{
		WindowStart: int(row.WindowStart),
		WindowEnd:   int(row.WindowEnd),
		Timezone:    row.Timezone,
		MaxRetries:  int(row.MaxRetries),
	}, nil
}

func (r *CallPolicyRepo) Upsert(ctx context.Context, orgID domain.OrgID, p domain.CallPolicy) error {
	return r.q.UpsertCallPolicy(ctx, gen.UpsertCallPolicyParams{
		OrgID:       string(orgID),
		WindowStart: int16(p.WindowStart),
		WindowEnd:   int16(p.WindowEnd),
		Timezone:    p.Timezone,
		MaxRetries:  int16(p.MaxRetries),
	})
}
