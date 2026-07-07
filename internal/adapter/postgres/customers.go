package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
)

type CustomerRepo struct{ q *gen.Queries }

func NewCustomerRepo(pool *pgxpool.Pool) *CustomerRepo { return &CustomerRepo{q: gen.New(pool)} }

func (r *CustomerRepo) Upsert(ctx context.Context, orgID domain.OrgID, phone, name, externalID string) (string, bool, error) {
	row, err := r.q.UpsertCustomer(ctx, gen.UpsertCustomerParams{
		ID: id.New("cust"), OrgID: string(orgID), Phone: phone,
		Name: strPtr(name), ExternalID: strPtr(externalID),
	})
	if err != nil {
		return "", false, err
	}
	return row.ID, row.Dnd, nil
}

func (r *CustomerRepo) ListByOrg(ctx context.Context, orgID domain.OrgID, limit int) ([]domain.Customer, error) {
	rows, err := r.q.ListCustomersByOrg(ctx, gen.ListCustomersByOrgParams{OrgID: string(orgID), Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Customer, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Customer{
			ID: row.ID, Phone: row.Phone, Name: deref(row.Name), ExternalID: deref(row.ExternalID),
			DND: row.Dnd, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (r *CustomerRepo) SetDND(ctx context.Context, orgID domain.OrgID, custID string, dnd bool) error {
	return r.q.SetCustomerDND(ctx, gen.SetCustomerDNDParams{ID: custID, OrgID: string(orgID), Dnd: dnd})
}
