// Package analytics composes the cross-channel dashboard rollup from the call,
// conversation, and credit stores.
package analytics

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type Service struct {
	calls   domain.CallRepo
	credits domain.CreditRepo
	repo    domain.AnalyticsRepo
}

func New(calls domain.CallRepo, credits domain.CreditRepo, repo domain.AnalyticsRepo) *Service {
	return &Service{calls: calls, credits: credits, repo: repo}
}

// Overview gathers voice, chat, and billing metrics for the org's dashboard.
func (s *Service) Overview(ctx context.Context, orgID domain.OrgID) (domain.AnalyticsOverview, error) {
	callSum, err := s.calls.Summary(ctx, orgID)
	if err != nil {
		return domain.AnalyticsOverview{}, err
	}
	convStats, err := s.repo.ConversationStats(ctx, orgID)
	if err != nil {
		return domain.AnalyticsOverview{}, err
	}
	bal, err := s.credits.Balance(ctx, orgID)
	if err != nil {
		return domain.AnalyticsOverview{}, err
	}
	spentToday, spent30d, err := s.repo.CreditSpend(ctx, orgID)
	if err != nil {
		return domain.AnalyticsOverview{}, err
	}
	return domain.AnalyticsOverview{
		Calls:         callSum,
		Conversations: convStats,
		Credits:       domain.CreditSummary{Balance: bal, SpentToday: spentToday, Spent30d: spent30d},
	}, nil
}
