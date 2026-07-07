// Package campaigns runs batch outbound calls over an org's callable customers.
package campaigns

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
)

type Service struct {
	repo      domain.CampaignRepo
	customers domain.CustomerRepo
	flows     domain.FlowRepo
	orch      domain.Orchestrator
}

func New(repo domain.CampaignRepo, customers domain.CustomerRepo, flows domain.FlowRepo, orch domain.Orchestrator) *Service {
	return &Service{repo: repo, customers: customers, flows: flows, orch: orch}
}

func (s *Service) Create(ctx context.Context, orgID domain.OrgID, name string) (string, error) {
	cid := id.New("camp")
	if err := s.repo.Create(ctx, orgID, cid, name); err != nil {
		return "", err
	}
	return cid, nil
}

func (s *Service) List(ctx context.Context, orgID domain.OrgID) ([]domain.Campaign, error) {
	return s.repo.ListByOrg(ctx, orgID)
}

func (s *Service) Get(ctx context.Context, orgID domain.OrgID, id string) (domain.Campaign, error) {
	c, owner, found, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	if !found || owner != orgID {
		return domain.Campaign{}, domain.ErrNotFound
	}
	return c, nil
}

// Start dispatches a confirmation call to every callable (non-DND) customer and
// marks the campaign completed. Returns how many calls were queued. No-op (with
// the existing count) if the campaign isn't a draft.
func (s *Service) Start(ctx context.Context, orgID domain.OrgID, id string) (int, error) {
	c, err := s.Get(ctx, orgID, id)
	if err != nil {
		return 0, err
	}
	if c.Status != "draft" {
		return c.TargetCount, nil
	}

	custs, err := s.customers.ListByOrg(ctx, orgID, 500)
	if err != nil {
		return 0, err
	}

	var flowID domain.FlowID
	if f, found, ferr := s.flows.GetActiveFlow(ctx, orgID, "order_confirm"); ferr == nil && found {
		flowID = f.ID
	}

	queued := 0
	for _, cust := range custs {
		if cust.DND {
			continue
		}
		if err := s.orch.EnqueuePlaceCall(ctx, domain.PlaceCallInput{OrgID: orgID, ToPhone: cust.Phone, FlowID: flowID}); err != nil {
			return queued, err
		}
		queued++
	}
	if err := s.repo.MarkStarted(ctx, orgID, id, queued); err != nil {
		return queued, err
	}
	return queued, nil
}
