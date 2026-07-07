// Package calls is the application service for voice interactions.
// It orchestrates the flow engine + a VoiceProvider + the repos, and depends
// only on domain interfaces.
package calls

import (
	"context"
	"encoding/json"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/flowengine"
	"github.com/ebnsina/yaver-api/pkg/id"
)

// creditsPerCall is what one call costs.
const creditsPerCall = 1

type Service struct {
	ivr      *flowengine.IVR
	provider domain.VoiceProvider
	outcomes domain.OutcomeRepo
	calls    domain.CallRepo
	flows    domain.FlowRepo
	credits  domain.CreditRepo
	clock    domain.Clock
}

func New(provider domain.VoiceProvider, outcomeRepo domain.OutcomeRepo, callRepo domain.CallRepo, flowRepo domain.FlowRepo, credits domain.CreditRepo, clock domain.Clock) *Service {
	return &Service{ivr: flowengine.NewIVR(), provider: provider, outcomes: outcomeRepo, calls: callRepo, flows: flowRepo, credits: credits, clock: clock}
}

// List returns recent calls for an org, newest first.
func (s *Service) List(ctx context.Context, orgID domain.OrgID, limit int) ([]domain.Call, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.calls.ListByOrg(ctx, orgID, limit)
}

// Summary returns the dashboard metrics rollup for an org.
func (s *Service) Summary(ctx context.Context, orgID domain.OrgID) (domain.CallSummary, error) {
	return s.calls.Summary(ctx, orgID)
}

// Get returns a single call scoped to the org (ErrNotFound if it belongs to
// another org, so nothing leaks).
func (s *Service) Get(ctx context.Context, orgID domain.OrgID, id domain.CallID) (*domain.Call, error) {
	c, err := s.calls.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

// RunTestCall loads the named active flow, drives it with a simulated keypress
// (the Phase 0 "no telco" path), persists the Call, and returns the outcome.
func (s *Service) RunTestCall(ctx context.Context, orgID domain.OrgID, toPhone, digit, flowName string) (*domain.Outcome, *domain.Call, error) {
	flow, found, err := s.flows.GetActiveFlow(ctx, orgID, flowName)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return nil, nil, domain.ErrNotFound
	}

	// Meter usage: a call costs credits. Charge before dialing.
	if ok, _, err := s.credits.TryDeduct(ctx, orgID, creditsPerCall, "call"); err != nil {
		return nil, nil, err
	} else if !ok {
		return nil, nil, domain.ErrInsufficientCredits
	}

	pid, err := s.provider.PlaceCall(ctx, domain.CallRequest{OrgID: orgID, ToPhone: toPhone, Flow: flow})
	if err != nil {
		return nil, nil, err
	}

	_, st, out := s.ivr.Start(flow)
	if out == nil { // reached a gather; simulate the customer's keypress
		_, out = s.ivr.OnDTMF(flow, st, digit)
	}

	call := &domain.Call{
		ID:             domain.CallID(id.New("call")),
		OrgID:          orgID,
		FlowID:         flow.ID,
		ProviderCallID: pid,
		Direction:      domain.Outbound,
		Status:         out.Status,
		Result:         out.Result,
		CreatedAt:      s.clock.Now(),
	}
	// Persist the call and the webhook outbox row in one transaction.
	if err := s.outcomes.RecordCallOutcome(ctx, call, outcomeEvent(call)); err != nil {
		return nil, nil, err
	}
	return out, call, nil
}

// outcomeEvent builds the webhook payload for a terminal call.
func outcomeEvent(c *domain.Call) *domain.OutboxEvent {
	event := "call." + string(c.Status) // e.g. call.completed, call.no_answer
	payload, _ := json.Marshal(map[string]any{
		"event":   event,
		"call_id": string(c.ID),
		"status":  string(c.Status),
		"result":  c.Result,
	})
	return &domain.OutboxEvent{Event: event, Payload: payload}
}

// PlaceCall is the place_call job handler: dial via the VoiceProvider and record
// a queued Call. (Live leg events land in Phase 1.) Written to be safe under the
// orchestrator's at-least-once delivery.
func (s *Service) PlaceCall(ctx context.Context, in domain.PlaceCallInput) error {
	// Meter usage before dialing; skip (no call) if the org is out of credits.
	if ok, _, err := s.credits.TryDeduct(ctx, in.OrgID, creditsPerCall, "call"); err != nil {
		return err
	} else if !ok {
		return domain.ErrInsufficientCredits
	}

	pid, err := s.provider.PlaceCall(ctx, domain.CallRequest{OrgID: in.OrgID, ToPhone: in.ToPhone})
	if err != nil {
		return err
	}
	// Queued call: persist only, no outbox event (not a terminal outcome).
	return s.outcomes.RecordCallOutcome(ctx, &domain.Call{
		ID:             domain.CallID(id.New("call")),
		OrgID:          in.OrgID,
		FlowID:         in.FlowID,
		ProviderCallID: pid,
		Direction:      domain.Outbound,
		Status:         domain.StatusQueued,
		CreatedAt:      s.clock.Now(),
	}, nil)
}
