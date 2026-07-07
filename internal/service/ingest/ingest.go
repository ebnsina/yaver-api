// Package ingest accepts inbound merchant events: normalize, dedup, and (for
// order_placed) enqueue a confirmation call — the Phase 1 wedge.
package ingest

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
	"github.com/ebnsina/yaver-api/pkg/phone"
)

type Service struct {
	events    domain.EventRepo
	customers domain.CustomerRepo
	flows     domain.FlowRepo
	orch      domain.Orchestrator
}

func New(events domain.EventRepo, customers domain.CustomerRepo, flows domain.FlowRepo, orch domain.Orchestrator) *Service {
	return &Service{events: events, customers: customers, flows: flows, orch: orch}
}

// Event is the normalized-at-transport input.
type Event struct {
	Type         string
	ExternalID   string
	Phone        string // raw; normalized here
	CustomerName string
	CustomerRef  string // merchant-side customer id
	Payload      []byte
}

// Accept stores the event idempotently and, when it's a new order_placed with a
// valid phone, enqueues a place_call against the active order_confirm flow.
func (s *Service) Accept(ctx context.Context, orgID domain.OrgID, ev Event) (eventID string, duplicate bool, err error) {
	var e164 string
	if ev.Phone != "" {
		e164, err = phone.NormalizeBD(ev.Phone)
		if err != nil {
			return "", false, domain.ErrInvalidPhone
		}
	}

	eventID = id.New("evt")
	inserted, err := s.events.Insert(ctx, domain.IngestEvent{
		ID:              eventID,
		OrgID:           string(orgID),
		Type:            ev.Type,
		ExternalEventID: ev.ExternalID,
		Phone:           e164,
		Payload:         ev.Payload,
	})
	if err != nil {
		return "", false, err
	}
	if !inserted {
		return "", true, nil // duplicate — idempotent no-op
	}

	// Upsert the customer and read their DND flag.
	var dnd bool
	if e164 != "" {
		if _, dnd, err = s.customers.Upsert(ctx, orgID, e164, ev.CustomerName, ev.CustomerRef); err != nil {
			return "", false, err
		}
	}

	// Trigger a confirmation call — unless the customer is on DND.
	if ev.Type == "order_placed" && e164 != "" && !dnd {
		var flowID domain.FlowID
		if f, found, ferr := s.flows.GetActiveFlow(ctx, orgID, "order_confirm"); ferr == nil && found {
			flowID = f.ID
		}
		if err := s.orch.EnqueuePlaceCall(ctx, domain.PlaceCallInput{OrgID: orgID, ToPhone: e164, FlowID: flowID}); err != nil {
			return "", false, err
		}
	}
	return eventID, false, nil
}
