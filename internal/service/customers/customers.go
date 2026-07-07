// Package customers lists customers and manages their DND (do-not-disturb) flag.
package customers

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type Service struct {
	repo domain.CustomerRepo
}

func New(repo domain.CustomerRepo) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, orgID domain.OrgID, limit int) ([]domain.Customer, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.ListByOrg(ctx, orgID, limit)
}

func (s *Service) SetDND(ctx context.Context, orgID domain.OrgID, id string, dnd bool) error {
	return s.repo.SetDND(ctx, orgID, id, dnd)
}
