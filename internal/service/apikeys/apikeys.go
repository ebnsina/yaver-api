// Package apikeys mints and authenticates merchant API keys.
package apikeys

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/apikey"
	"github.com/ebnsina/yaver-api/pkg/id"
)

type Service struct {
	repo domain.APIKeyRepo
}

func New(repo domain.APIKeyRepo) *Service { return &Service{repo: repo} }

// Mint creates a secret key for an org and returns the full key exactly once.
func (s *Service) Mint(ctx context.Context, orgID domain.OrgID, name string) (fullKey string, err error) {
	full, prefix, hash := apikey.Generate()
	if err := s.repo.Create(ctx, id.New("key"), string(orgID), prefix, hash, name, "secret"); err != nil {
		return "", err
	}
	return full, nil
}

// MintPublishable creates a publishable (yvr_pk_) key, safe to embed client-side.
func (s *Service) MintPublishable(ctx context.Context, orgID domain.OrgID, name string) (fullKey string, err error) {
	full, prefix, hash := apikey.GeneratePublishable()
	if err := s.repo.Create(ctx, id.New("key"), string(orgID), prefix, hash, name, "publishable"); err != nil {
		return "", err
	}
	return full, nil
}

// List returns public metadata for an org's keys (never secrets).
func (s *Service) List(ctx context.Context, orgID domain.OrgID) ([]domain.APIKeyInfo, error) {
	return s.repo.ListByOrg(ctx, string(orgID))
}

// Authenticate resolves a presented key to its org and kind. ok=false when the
// key is malformed, unknown, or the hash doesn't match.
func (s *Service) Authenticate(ctx context.Context, presented string) (orgID domain.OrgID, kind string, ok bool, err error) {
	prefix, valid := apikey.Prefix(presented)
	if !valid {
		return "", "", false, nil
	}
	keyID, org, k, hash, found, err := s.repo.ByPrefix(ctx, prefix)
	if err != nil {
		return "", "", false, err
	}
	if !found || !apikey.Verify(presented, hash) {
		return "", "", false, nil
	}
	_ = s.repo.Touch(ctx, keyID) // best-effort last-used
	return domain.OrgID(org), k, true, nil
}
