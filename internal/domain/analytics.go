package domain

import "context"

// AnalyticsOverview is the cross-channel dashboard rollup: voice, chat, and
// billing in one read. The "money, not metrics" home screen.
type AnalyticsOverview struct {
	Calls         CallSummary
	Conversations ConversationStats
	Credits       CreditSummary
}

// ConversationStats aggregates conversations plus the AI-enriched outcome mix.
type ConversationStats struct {
	Total      int
	Today      int
	Resolved   int
	Pending    int
	Sale       int
	NeedsHuman int
}

// CreditSummary is the org's balance plus recent spend.
type CreditSummary struct {
	Balance    int
	SpentToday int
	Spent30d   int
}

// AnalyticsRepo supplies the cross-channel aggregates the CallRepo/CreditRepo
// don't already cover.
type AnalyticsRepo interface {
	ConversationStats(ctx context.Context, orgID OrgID) (ConversationStats, error)
	// CreditSpend returns credits consumed today and over the last 30 days.
	CreditSpend(ctx context.Context, orgID OrgID) (today, last30d int, err error)
}
