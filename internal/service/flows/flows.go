// Package flows serves and edits flow definitions for the no-code builder.
package flows

import (
	"context"
	"encoding/json"

	"github.com/ebnsina/yaver-api/internal/domain"
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
