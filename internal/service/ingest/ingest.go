// Package ingest accepts inbound merchant events: normalize, dedup, and route
// each known event type to the flow that handles it (order confirmation,
// cart recovery, delivery reminder).
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

// eventFlow maps an inbound event type to the flow name that handles it. Types
// not listed are still stored (for audit/analytics) but trigger no call.
var eventFlow = map[string]string{
	"order_placed":     "order_confirm",
	"abandoned_cart":   "cart_recovery",
	"out_for_delivery": "delivery_reminder",
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

	// Route the event to its flow and place a call — unless the customer is on
	// DND, the phone is missing, or the merchant hasn't built a flow for it.
	if flowName, mapped := eventFlow[ev.Type]; mapped && e164 != "" && !dnd {
		if f, found, ferr := s.flows.GetActiveFlow(ctx, orgID, flowName); ferr != nil {
			return "", false, ferr
		} else if found {
			if err := s.orch.EnqueuePlaceCall(ctx, domain.PlaceCallInput{OrgID: orgID, ToPhone: e164, FlowID: f.ID}); err != nil {
				return "", false, err
			}
		}
	}
	return eventID, false, nil
}
