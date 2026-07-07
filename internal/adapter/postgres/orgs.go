package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

type OrgRepo struct{ pool *pgxpool.Pool }

func NewOrgRepo(pool *pgxpool.Pool) *OrgRepo { return &OrgRepo{pool: pool} }

// EnsureForUser returns the user's org id, creating the org (and seeding a
// default order-confirmation flow) on first call. Idempotent and race-safe via
// the unique index on orgs.owner_user_id.
func (r *OrgRepo) EnsureForUser(ctx context.Context, userID, name string) (domain.OrgID, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	q := gen.New(r.pool)

	id, err := q.GetOrgByOwner(ctx, uid)
	if err == nil {
		return domain.OrgID(id.String()), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	orgID := uuid.New()
	if err := q.CreateOrg(ctx, gen.CreateOrgParams{ID: orgID, Name: name, OwnerUserID: uid}); err != nil {
		return "", err
	}
	// Re-read: another concurrent request may have won the insert.
	id, err = q.GetOrgByOwner(ctx, uid)
	if err != nil {
		return "", err
	}
	if id == orgID { // we created it — seed the default flow
		_ = q.CreateFlow(ctx, gen.CreateFlowParams{
			ID:      "flow_" + orgID.String(),
			OrgID:   orgID.String(),
			Name:    "order_confirm",
			Version: 1,
			Channel: "voice",
			Type:    "ivr",
			Locale:  "en",
			Spec:    defaultOrderConfirmSpec,
		})
	}
	return domain.OrgID(id.String()), nil
}

// defaultOrderConfirmSpec is the IVR flow every new org starts with (editable in
// the no-code builder later).
var defaultOrderConfirmSpec = []byte(`{
  "entry": "greet",
  "nodes": {
    "greet":      {"say": {"tts": "Press 1 to confirm your order, 2 to cancel, 3 to reschedule."},
                   "gather": {"digits": 1, "timeout_s": 6},
                   "on": {"1": "confirmed", "2": "cancelled", "3": "reschedule", "timeout": "no_input"}},
    "confirmed":  {"say": {"audio": "confirmed.wav"},  "result": "confirmed",  "end": true},
    "cancelled":  {"say": {"audio": "cancelled.wav"},  "result": "cancelled",  "end": true},
    "reschedule": {"say": {"audio": "reschedule.wav"}, "result": "reschedule", "end": true},
    "no_input":   {"result": "no_answer", "end": true}
  }
}`)
