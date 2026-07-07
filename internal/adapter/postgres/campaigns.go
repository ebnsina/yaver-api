package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

type CampaignRepo struct{ q *gen.Queries }

func NewCampaignRepo(pool *pgxpool.Pool) *CampaignRepo { return &CampaignRepo{q: gen.New(pool)} }

func (r *CampaignRepo) Create(ctx context.Context, orgID domain.OrgID, id, name string) error {
	return r.q.CreateCampaign(ctx, gen.CreateCampaignParams{ID: id, OrgID: string(orgID), Name: name})
}

func (r *CampaignRepo) ListByOrg(ctx context.Context, orgID domain.OrgID) ([]domain.Campaign, error) {
	rows, err := r.q.ListCampaignsByOrg(ctx, string(orgID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.Campaign, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Campaign{
			ID: row.ID, Name: row.Name, Status: row.Status, TargetCount: int(row.TargetCount),
			CreatedAt: row.CreatedAt, StartedAt: row.StartedAt,
		})
	}
	return out, nil
}

func (r *CampaignRepo) Get(ctx context.Context, id string) (domain.Campaign, domain.OrgID, bool, error) {
	row, err := r.q.GetCampaign(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Campaign{}, "", false, nil
	}
	if err != nil {
		return domain.Campaign{}, "", false, err
	}
	return domain.Campaign{
		ID: row.ID, Name: row.Name, Status: row.Status, TargetCount: int(row.TargetCount),
		CreatedAt: row.CreatedAt, StartedAt: row.StartedAt,
	}, domain.OrgID(row.OrgID), true, nil
}

func (r *CampaignRepo) MarkStarted(ctx context.Context, orgID domain.OrgID, id string, targetCount int) error {
	return r.q.StartCampaign(ctx, gen.StartCampaignParams{ID: id, OrgID: string(orgID), TargetCount: int32(targetCount)})
}
