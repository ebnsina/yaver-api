package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
)

type FlowRepo struct {
	q *gen.Queries
}

func NewFlowRepo(pool *pgxpool.Pool) *FlowRepo { return &FlowRepo{q: gen.New(pool)} }

func (r *FlowRepo) GetActiveFlow(ctx context.Context, orgID domain.OrgID, name string) (domain.Flow, bool, error) {
	row, err := r.q.GetActiveFlowByName(ctx, gen.GetActiveFlowByNameParams{OrgID: string(orgID), Name: name})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Flow{}, false, nil
	}
	if err != nil {
		return domain.Flow{}, false, err
	}

	f := domain.Flow{
		ID:      domain.FlowID(row.ID),
		OrgID:   domain.OrgID(row.OrgID),
		Name:    row.Name,
		Version: int(row.Version),
		Channel: domain.Channel(row.Channel),
		Type:    domain.FlowType(row.Type),
		Locale:  row.Locale,
	}
	if f.Type == domain.FlowIVR {
		if err := json.Unmarshal(row.Spec, &f.IVR); err != nil {
			return domain.Flow{}, false, err
		}
	}
	return f, true, nil
}

func (r *FlowRepo) Create(ctx context.Context, orgID domain.OrgID, nf domain.NewFlow) (domain.FlowID, error) {
	fid := id.New("flow")
	err := r.q.CreateFlow(ctx, gen.CreateFlowParams{
		ID:      fid,
		OrgID:   string(orgID),
		Name:    nf.Name,
		Version: 1,
		Channel: string(nf.Channel),
		Type:    string(nf.Type),
		Locale:  nf.Locale,
		Spec:    nf.Spec,
	})
	if err != nil {
		return "", err
	}
	return domain.FlowID(fid), nil
}

func (r *FlowRepo) ListByOrg(ctx context.Context, orgID domain.OrgID) ([]domain.FlowSummary, error) {
	rows, err := r.q.ListFlowsByOrg(ctx, string(orgID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.FlowSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.FlowSummary{
			ID: domain.FlowID(row.ID), Name: row.Name, Version: int(row.Version),
			Channel: domain.Channel(row.Channel), Type: domain.FlowType(row.Type), Active: row.IsActive,
		})
	}
	return out, nil
}

func (r *FlowRepo) GetByID(ctx context.Context, id domain.FlowID) (domain.FlowDetail, bool, error) {
	row, err := r.q.GetFlowByID(ctx, string(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.FlowDetail{}, false, nil
	}
	if err != nil {
		return domain.FlowDetail{}, false, err
	}
	return domain.FlowDetail{
		ID: domain.FlowID(row.ID), OrgID: domain.OrgID(row.OrgID), Name: row.Name, Version: int(row.Version),
		Channel: domain.Channel(row.Channel), Type: domain.FlowType(row.Type), Locale: row.Locale, Spec: row.Spec,
	}, true, nil
}

func (r *FlowRepo) UpdateSpec(ctx context.Context, id domain.FlowID, orgID domain.OrgID, spec []byte) error {
	return r.q.UpdateFlowSpec(ctx, gen.UpdateFlowSpecParams{ID: string(id), OrgID: string(orgID), Spec: spec})
}
