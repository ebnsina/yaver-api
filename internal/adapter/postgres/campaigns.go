package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
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
			CreatedAt: row.CreatedAt, StartedAt: row.StartedAt, ScheduledAt: row.ScheduledAt,
		})
	}
	return out, nil
}

func (r *CampaignRepo) Get(ctx context.Context, cid string) (domain.Campaign, domain.OrgID, bool, error) {
	row, err := r.q.GetCampaign(ctx, cid)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Campaign{}, "", false, nil
	}
	if err != nil {
		return domain.Campaign{}, "", false, err
	}
	return domain.Campaign{
		ID: row.ID, Name: row.Name, Status: row.Status, TargetCount: int(row.TargetCount),
		CreatedAt: row.CreatedAt, StartedAt: row.StartedAt, ScheduledAt: row.ScheduledAt,
	}, domain.OrgID(row.OrgID), true, nil
}

func (r *CampaignRepo) MarkStarted(ctx context.Context, orgID domain.OrgID, id string, targetCount int) error {
	return r.q.StartCampaign(ctx, gen.StartCampaignParams{ID: id, OrgID: string(orgID), TargetCount: int32(targetCount)})
}

func (r *CampaignRepo) AddRecipient(ctx context.Context, orgID domain.OrgID, campaignID string, rec domain.Recipient) error {
	return r.q.AddRecipient(ctx, gen.AddRecipientParams{
		ID: id.New("rcp"), CampaignID: campaignID, OrgID: string(orgID), Phone: rec.Phone, Name: rec.Name,
	})
}

func (r *CampaignRepo) Recipients(ctx context.Context, campaignID string) ([]domain.Recipient, error) {
	rows, err := r.q.ListRecipients(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Recipient, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Recipient{Phone: row.Phone, Name: row.Name})
	}
	return out, nil
}

func (r *CampaignRepo) Schedule(ctx context.Context, orgID domain.OrgID, id string, at time.Time) error {
	return r.q.ScheduleCampaign(ctx, gen.ScheduleCampaignParams{ID: id, OrgID: string(orgID), ScheduledAt: &at})
}

func (r *CampaignRepo) DueCampaigns(ctx context.Context, now time.Time) ([]domain.CampaignRef, error) {
	rows, err := r.q.DueCampaigns(ctx, &now)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CampaignRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CampaignRef{ID: row.ID, OrgID: domain.OrgID(row.OrgID)})
	}
	return out, nil
}
