package domain

import (
	"context"
	"time"
)

// Campaign is a batch outbound-call job over the org's callable customers.
type Campaign struct {
	ID          string
	Name        string
	Status      string // draft | completed
	TargetCount int
	CreatedAt   time.Time
	StartedAt   *time.Time
}

// CampaignRepo persists campaigns.
type CampaignRepo interface {
	Create(ctx context.Context, orgID OrgID, id, name string) error
	ListByOrg(ctx context.Context, orgID OrgID) ([]Campaign, error)
	// Get returns the campaign and its owning org (caller checks ownership).
	Get(ctx context.Context, id string) (c Campaign, orgID OrgID, found bool, err error)
	// MarkStarted flips a draft to completed with the dispatched count (org-scoped).
	MarkStarted(ctx context.Context, orgID OrgID, id string, targetCount int) error
}
