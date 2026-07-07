package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

type AnalyticsRepo struct{ q *gen.Queries }

func NewAnalyticsRepo(pool *pgxpool.Pool) *AnalyticsRepo { return &AnalyticsRepo{q: gen.New(pool)} }

func (r *AnalyticsRepo) ConversationStats(ctx context.Context, orgID domain.OrgID) (domain.ConversationStats, error) {
	row, err := r.q.ConversationStats(ctx, string(orgID))
	if err != nil {
		return domain.ConversationStats{}, err
	}
	return domain.ConversationStats{
		Total:      int(row.Total),
		Today:      int(row.Today),
		Resolved:   int(row.Resolved),
		Pending:    int(row.Pending),
		Sale:       int(row.Sale),
		NeedsHuman: int(row.NeedsHuman),
	}, nil
}

func (r *AnalyticsRepo) CreditSpend(ctx context.Context, orgID domain.OrgID) (int, int, error) {
	row, err := r.q.CreditSpend(ctx, string(orgID))
	if err != nil {
		return 0, 0, err
	}
	return int(row.Today), int(row.Last30d), nil
}
