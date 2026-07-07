package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

// APIKeyRepo ------------------------------------------------------------------

type APIKeyRepo struct{ q *gen.Queries }

func NewAPIKeyRepo(pool *pgxpool.Pool) *APIKeyRepo { return &APIKeyRepo{q: gen.New(pool)} }

func (r *APIKeyRepo) Create(ctx context.Context, id, orgID, prefix string, secretHash []byte, name string) error {
	return r.q.CreateAPIKey(ctx, gen.CreateAPIKeyParams{
		ID: id, OrgID: orgID, Prefix: prefix, SecretHash: secretHash, Name: strPtr(name),
	})
}

func (r *APIKeyRepo) ByPrefix(ctx context.Context, prefix string) (id, orgID string, secretHash []byte, found bool, err error) {
	row, err := r.q.GetAPIKeyByPrefix(ctx, prefix)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil, false, nil
	}
	if err != nil {
		return "", "", nil, false, err
	}
	return row.ID, row.OrgID, row.SecretHash, true, nil
}

func (r *APIKeyRepo) Touch(ctx context.Context, id string) error {
	return r.q.TouchAPIKey(ctx, id)
}

func (r *APIKeyRepo) ListByOrg(ctx context.Context, orgID string) ([]domain.APIKeyInfo, error) {
	rows, err := r.q.ListAPIKeysByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.APIKeyInfo, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.APIKeyInfo{
			Prefix:     row.Prefix,
			Name:       deref(row.Name),
			CreatedAt:  row.CreatedAt,
			LastUsedAt: row.LastUsedAt,
		})
	}
	return out, nil
}

// EventRepo -------------------------------------------------------------------

type EventRepo struct{ q *gen.Queries }

func NewEventRepo(pool *pgxpool.Pool) *EventRepo { return &EventRepo{q: gen.New(pool)} }

func (r *EventRepo) Insert(ctx context.Context, e domain.IngestEvent) (bool, error) {
	_, err := r.q.InsertEvent(ctx, gen.InsertEventParams{
		ID:              e.ID,
		OrgID:           e.OrgID,
		Type:            e.Type,
		ExternalEventID: e.ExternalEventID,
		Payload:         e.Payload,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // ON CONFLICT DO NOTHING -> duplicate
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
