// Package calls is the application service for voice interactions.
// It orchestrates the flow engine + a VoiceProvider + the call repo, and
// depends only on domain interfaces.
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
	repo     domain.CallRepo
	clock    domain.Clock
}

func New(provider domain.VoiceProvider, repo domain.CallRepo, clock domain.Clock) *Service {
	return &Service{ivr: flowengine.NewIVR(), provider: provider, repo: repo, clock: clock}
}

// RunTestCall drives a full IVR interaction with a simulated keypress — the
// Phase 0 "flow logic, no telco" path. It places the call via the (mock)
// provider, advances the engine, persists the Call, and returns the outcome.
func (s *Service) RunTestCall(ctx context.Context, orgID domain.OrgID, toPhone, digit string, flow domain.Flow) (*domain.Outcome, *domain.Call, error) {
	if _, err := s.provider.PlaceCall(ctx, domain.CallRequest{
		OrgID:   orgID,
		ToPhone: toPhone,
		Flow:    flow,
	}); err != nil {
		return nil, nil, err
	}

	_, st, out := s.ivr.Start(flow)
	if out == nil { // reached a gather; simulate the customer's keypress
		_, out = s.ivr.OnDTMF(flow, st, digit)
	}

	call := &domain.Call{
		ID:        domain.CallID(id.New("call")),
		OrgID:     orgID,
		FlowID:    flow.ID,
		Direction: domain.Outbound,
		Status:    out.Status,
		Result:    out.Result,
		CreatedAt: s.clock.Now(),
	}
	if err := s.repo.Create(ctx, call); err != nil {
		return nil, nil, err
	}
	return out, call, nil
}
