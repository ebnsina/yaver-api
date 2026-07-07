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

type ChannelRepo struct{ q *gen.Queries }

func NewChannelRepo(pool *pgxpool.Pool) *ChannelRepo { return &ChannelRepo{q: gen.New(pool)} }

func (r *ChannelRepo) Upsert(ctx context.Context, orgID domain.OrgID, typ, externalID string, encToken []byte, verifyToken string) error {
	return r.q.UpsertChannel(ctx, gen.UpsertChannelParams{
		ID: id.New("chan"), OrgID: string(orgID), Type: typ,
		ExternalID: externalID, AccessToken: encToken, VerifyToken: verifyToken,
	})
}

func (r *ChannelRepo) List(ctx context.Context, orgID domain.OrgID) ([]domain.ChannelConnection, error) {
	rows, err := r.q.ListChannelsByOrg(ctx, string(orgID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.ChannelConnection, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ChannelConnection{Type: row.Type, ExternalID: row.ExternalID, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (r *ChannelRepo) Delete(ctx context.Context, orgID domain.OrgID, typ string) error {
	return r.q.DeleteChannel(ctx, gen.DeleteChannelParams{OrgID: string(orgID), Type: typ})
}

func (r *ChannelRepo) ByExternalID(ctx context.Context, externalID string) (domain.OrgID, string, []byte, string, bool, error) {
	row, err := r.q.GetChannelByExternalID(ctx, externalID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil, "", false, nil
	}
	if err != nil {
		return "", "", nil, "", false, err
	}
	return domain.OrgID(row.OrgID), row.Type, row.AccessToken, row.VerifyToken, true, nil
}

func (r *ChannelRepo) OrgForVerifyToken(ctx context.Context, verifyToken string) (bool, error) {
	_, err := r.q.GetChannelByVerifyToken(ctx, verifyToken)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
