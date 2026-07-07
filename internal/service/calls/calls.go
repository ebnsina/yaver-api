// Package calls is the application service for voice interactions.
// It orchestrates the flow engine + a VoiceProvider + the repos, and depends
// only on domain interfaces.
package calls

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/flowengine"
	"github.com/ebnsina/yaver-api/pkg/id"
)

type Service struct {
	ivr      *flowengine.IVR
	provider domain.VoiceProvider
	calls    domain.CallRepo
	flows    domain.FlowRepo
	clock    domain.Clock
}

func New(provider domain.VoiceProvider, callRepo domain.CallRepo, flowRepo domain.FlowRepo, clock domain.Clock) *Service {
	return &Service{ivr: flowengine.NewIVR(), provider: provider, calls: callRepo, flows: flowRepo, clock: clock}
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
	if err := s.calls.Create(ctx, call); err != nil {
		return nil, nil, err
	}
	return out, call, nil
}

// PlaceCall is the place_call job handler: dial via the VoiceProvider and record
// a queued Call. (Live leg events land in Phase 1.) Written to be safe under the
// orchestrator's at-least-once delivery.
func (s *Service) PlaceCall(ctx context.Context, in domain.PlaceCallInput) error {
	pid, err := s.provider.PlaceCall(ctx, domain.CallRequest{OrgID: in.OrgID, ToPhone: in.ToPhone})
	if err != nil {
		return err
	}
	return s.calls.Create(ctx, &domain.Call{
		ID:             domain.CallID(id.New("call")),
		OrgID:          in.OrgID,
		FlowID:         in.FlowID,
		ProviderCallID: pid,
		Direction:      domain.Outbound,
		Status:         domain.StatusQueued,
		CreatedAt:      s.clock.Now(),
	})
}
