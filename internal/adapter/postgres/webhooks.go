package postgres

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

// OutcomeRepo writes a call + optional outbox row atomically. -----------------

type OutcomeRepo struct{ pool *pgxpool.Pool }

func NewOutcomeRepo(pool *pgxpool.Pool) *OutcomeRepo { return &OutcomeRepo{pool: pool} }

func (r *OutcomeRepo) RecordCallOutcome(ctx context.Context, c *domain.Call, outbox *domain.OutboxEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := gen.New(tx)
	if err := q.CreateCall(ctx, gen.CreateCallParams{
		ID:             string(c.ID),
		OrgID:          string(c.OrgID),
		FlowID:         strPtr(string(c.FlowID)),
		ProviderCallID: strPtr(string(c.ProviderCallID)),
		Direction:      string(c.Direction),
		Status:         string(c.Status),
		Result:         strPtr(c.Result),
	}); err != nil {
		return err
	}
	if outbox != nil {
		if err := q.InsertOutbox(ctx, gen.InsertOutboxParams{
			OrgID:   string(c.OrgID),
			Event:   outbox.Event,
			Payload: outbox.Payload,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// WebhookRepo -----------------------------------------------------------------

type WebhookRepo struct{ pool *pgxpool.Pool }

func NewWebhookRepo(pool *pgxpool.Pool) *WebhookRepo { return &WebhookRepo{pool: pool} }

func (r *WebhookRepo) UpsertEndpoint(ctx context.Context, id, orgID, url string, secretEnc []byte, events []string) error {
	return gen.New(r.pool).UpsertWebhookEndpoint(ctx, gen.UpsertWebhookEndpointParams{
		ID: id, OrgID: orgID, Url: url, SecretEnc: secretEnc, Events: events,
	})
}

func (r *WebhookRepo) GetEndpoint(ctx context.Context, orgID string) (domain.WebhookEndpointRow, bool, error) {
	row, err := gen.New(r.pool).GetWebhookEndpoint(ctx, orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WebhookEndpointRow{}, false, nil
	}
	if err != nil {
		return domain.WebhookEndpointRow{}, false, err
	}
	return domain.WebhookEndpointRow{
		OrgID: row.OrgID, URL: row.Url, SecretEnc: row.SecretEnc, Events: row.Events, Active: row.Active,
	}, true, nil
}

// DrainOutbox: claim → create deliveries for subscribed endpoints → mark
// dispatched, in one transaction so the SKIP LOCKED lock is held throughout.
func (r *WebhookRepo) DrainOutbox(ctx context.Context, limit int, newDeliveryID func() string) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	q := gen.New(tx)

	rows, err := q.ClaimUndispatchedOutbox(ctx, int32(limit))
	if err != nil {
		return 0, err
	}
	created := 0
	for _, row := range rows {
		ep, epErr := q.GetWebhookEndpoint(ctx, row.OrgID)
		subscribed := epErr == nil && ep.Active && (len(ep.Events) == 0 || slices.Contains(ep.Events, row.Event))
		if subscribed {
			if err := q.CreateDelivery(ctx, gen.CreateDeliveryParams{
				ID: newDeliveryID(), OrgID: row.OrgID, Event: row.Event, Url: ep.Url, Payload: row.Payload,
			}); err != nil {
				return created, err
			}
			created++
		}
		if err := q.MarkOutboxDispatched(ctx, row.ID); err != nil {
			return created, err
		}
	}
	return created, tx.Commit(ctx)
}

func (r *WebhookRepo) DueDeliveries(ctx context.Context, limit int) ([]domain.DueDelivery, error) {
	rows, err := gen.New(r.pool).DueDeliveries(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]domain.DueDelivery, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.DueDelivery{
			ID: row.ID, OrgID: row.OrgID, Event: row.Event, URL: row.Url, Payload: row.Payload, Attempts: int(row.Attempts),
		})
	}
	return out, nil
}

func (r *WebhookRepo) MarkDelivered(ctx context.Context, id string, statusCode int) error {
	return gen.New(r.pool).MarkDelivered(ctx, gen.MarkDeliveredParams{ID: id, LastStatusCode: int4(statusCode)})
}

func (r *WebhookRepo) Reschedule(ctx context.Context, id string, statusCode int, errMsg, status string, nextRetryAt time.Time) error {
	return gen.New(r.pool).RescheduleDelivery(ctx, gen.RescheduleDeliveryParams{
		ID: id, LastStatusCode: int4(statusCode), LastError: strPtr(errMsg), Status: status, NextRetryAt: nextRetryAt,
	})
}

func int4(v int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(v), Valid: true}
}
