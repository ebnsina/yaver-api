// Package flows serves and edits flow definitions for the no-code builder.
package flows

import (
	"context"
	"encoding/json"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/flowengine"
)

type Service struct {
	repo domain.FlowRepo
}

func New(repo domain.FlowRepo) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, orgID domain.OrgID) ([]domain.FlowSummary, error) {
	return s.repo.ListByOrg(ctx, orgID)
}

// Get returns a flow scoped to the org (ErrNotFound otherwise — no leak).
func (s *Service) Get(ctx context.Context, orgID domain.OrgID, id domain.FlowID) (domain.FlowDetail, error) {
	fd, found, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, err
	}
	if !found || fd.OrgID != orgID {
		return domain.FlowDetail{}, domain.ErrNotFound
	}
	return fd, nil
}

// knownFlowTypes are the flow types the builder can create today.
var knownFlowTypes = map[domain.FlowType]bool{domain.FlowIVR: true, domain.FlowChat: true}

// Create adds a new active flow for the org. Names are unique per org among
// active flows so event routing (name → flow) stays unambiguous.
func (s *Service) Create(ctx context.Context, orgID domain.OrgID, nf domain.NewFlow) (domain.FlowID, error) {
	if nf.Name == "" || !knownFlowTypes[nf.Type] {
		return "", domain.ErrFlowInvalid
	}
	if nf.Locale == "" {
		nf.Locale = "en"
	}
	if nf.Channel == "" {
		if nf.Type == domain.FlowChat {
			nf.Channel = domain.ChannelChat
		} else {
			nf.Channel = domain.ChannelVoice
		}
	}
	// IVR flows carry a keypad spec we can validate up front.
	if nf.Type == domain.FlowIVR {
		if err := validateIVR(nf.Spec); err != nil {
			return "", err
		}
	}
	// Reject a duplicate active name so routing stays unambiguous.
	if _, found, err := s.repo.GetActiveFlow(ctx, orgID, nf.Name); err != nil {
		return "", err
	} else if found {
		return "", domain.ErrConflict
	}
	return s.repo.Create(ctx, orgID, nf)
}

// UpdateSpec validates and replaces an IVR flow's spec.
func (s *Service) UpdateSpec(ctx context.Context, orgID domain.OrgID, id domain.FlowID, spec []byte) error {
	// Ownership check.
	if _, err := s.Get(ctx, orgID, id); err != nil {
		return err
	}
	if err := validateIVR(spec); err != nil {
		return err
	}
	return s.repo.UpdateSpec(ctx, id, orgID, spec)
}

// Simulate dry-runs an IVR spec against a sequence of keypad inputs (digits or
// "timeout") and returns the full trace + outcome — the builder's in-browser
// simulator. Pure: no telephony, no credits, no persistence.
func (s *Service) Simulate(spec []byte, inputs []string) (flowengine.Simulation, error) {
	if err := validateIVR(spec); err != nil {
		return flowengine.Simulation{}, err
	}
	var ivr domain.IVRSpec
	if err := json.Unmarshal(spec, &ivr); err != nil {
		return flowengine.Simulation{}, domain.ErrFlowInvalid
	}
	f := domain.Flow{Type: domain.FlowIVR, IVR: ivr}
	return flowengine.NewIVR().Simulate(f, inputs), nil
}

// validateIVR ensures the spec parses and its entry node exists.
func validateIVR(raw []byte) error {
	var spec domain.IVRSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return domain.ErrFlowInvalid
	}
	if spec.Entry == "" {
		return domain.ErrFlowInvalid
	}
	if _, ok := spec.Nodes[spec.Entry]; !ok {
		return domain.ErrFlowInvalid
	}
	return nil
}
