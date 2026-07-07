package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

type InsightRepo struct{ q *gen.Queries }

func NewInsightRepo(pool *pgxpool.Pool) *InsightRepo { return &InsightRepo{q: gen.New(pool)} }

func (r *InsightRepo) Save(ctx context.Context, orgID domain.OrgID, conversationID string, in domain.ConversationInsight) error {
	return r.q.UpsertInsight(ctx, gen.UpsertInsightParams{
		ConversationID: conversationID,
		OrgID:          string(orgID),
		Summary:        in.Summary,
		Outcome:        in.Outcome,
		Sentiment:      in.Sentiment,
		NextAction:     in.NextAction,
	})
}

func (r *InsightRepo) Get(ctx context.Context, conversationID string) (domain.ConversationInsight, bool, error) {
	row, err := r.q.GetInsight(ctx, conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ConversationInsight{}, false, nil
	}
	if err != nil {
		return domain.ConversationInsight{}, false, err
	}
	return domain.ConversationInsight{
		Summary:    row.Summary,
		Outcome:    row.Outcome,
		Sentiment:  row.Sentiment,
		NextAction: row.NextAction,
		CreatedAt:  row.CreatedAt,
	}, true, nil
}
