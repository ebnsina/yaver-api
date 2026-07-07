package domain

import (
	"context"
	"time"
)

// Campaign is a batch outbound-call job. It targets an imported recipient list,
// or (when none is imported) all of the org's callable customers.
type Campaign struct {
	ID          string
	Name        string
	Status      string // draft | scheduled | completed
	TargetCount int
	CreatedAt   time.Time
	StartedAt   *time.Time
	ScheduledAt *time.Time
}

// Recipient is one target of a campaign (imported from CSV).
type Recipient struct {
	Phone string
	Name  string
}

// CampaignRef identifies a campaign and its owning org (for the scheduler sweep).
type CampaignRef struct {
	ID    string
	OrgID OrgID
}

// CampaignRepo persists campaigns and their recipient lists.
type CampaignRepo interface {
	Create(ctx context.Context, orgID OrgID, id, name string) error
	ListByOrg(ctx context.Context, orgID OrgID) ([]Campaign, error)
	// Get returns the campaign and its owning org (caller checks ownership).
	Get(ctx context.Context, id string) (c Campaign, orgID OrgID, found bool, err error)
	// MarkStarted flips a draft/scheduled campaign to completed with the count.
	MarkStarted(ctx context.Context, orgID OrgID, id string, targetCount int) error

	// AddRecipient inserts one recipient, ignoring duplicates within the campaign.
	AddRecipient(ctx context.Context, orgID OrgID, campaignID string, r Recipient) error
	Recipients(ctx context.Context, campaignID string) ([]Recipient, error)
	// Schedule moves a draft campaign to 'scheduled' with a fire time (org-scoped).
	Schedule(ctx context.Context, orgID OrgID, id string, at time.Time) error
	// DueCampaigns lists scheduled campaigns whose fire time has arrived.
	DueCampaigns(ctx context.Context, now time.Time) ([]CampaignRef, error)
}
