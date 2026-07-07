package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

type ChatSettingsRepo struct{ q *gen.Queries }

func NewChatSettingsRepo(pool *pgxpool.Pool) *ChatSettingsRepo {
	return &ChatSettingsRepo{q: gen.New(pool)}
}

func (r *ChatSettingsRepo) Get(ctx context.Context, orgID domain.OrgID) (domain.ChatSettings, error) {
	row, err := r.q.GetChatSettings(ctx, string(orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DefaultChatSettings(), nil
	}
	if err != nil {
		return domain.ChatSettings{}, err
	}
	return domain.ChatSettings{
		Instructions: row.Instructions, WidgetTitle: row.WidgetTitle,
		Welcome: row.Welcome, Accent: row.Accent,
	}, nil
}

func (r *ChatSettingsRepo) Upsert(ctx context.Context, orgID domain.OrgID, s domain.ChatSettings) error {
	return r.q.UpsertChatSettings(ctx, gen.UpsertChatSettingsParams{
		OrgID: string(orgID), Instructions: s.Instructions,
		WidgetTitle: s.WidgetTitle, Welcome: s.Welcome, Accent: s.Accent,
	})
}
