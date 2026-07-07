package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
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
