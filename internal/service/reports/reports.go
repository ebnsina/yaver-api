// Package reports answers natural-language questions over an org's metrics — the
// "AI custom reports" feature. It gathers the analytics context and hands it to
// a provider-agnostic domain.Reporter.
package reports

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/analytics"
)

type Service struct {
	analytics *analytics.Service
	reporter  domain.Reporter
}

func New(analyticsSvc *analytics.Service, reporter domain.Reporter) *Service {
	return &Service{analytics: analyticsSvc, reporter: reporter}
}

// Ask answers a question against the org's current metrics and returns both the
// answer text and the data it was grounded in (so the UI can show the numbers).
func (s *Service) Ask(ctx context.Context, orgID domain.OrgID, question string) (string, domain.AnalyticsOverview, error) {
	data, err := s.analytics.Overview(ctx, orgID)
	if err != nil {
		return "", domain.AnalyticsOverview{}, err
	}
	answer, err := s.reporter.Answer(ctx, question, data)
	if err != nil {
		return "", domain.AnalyticsOverview{}, err
	}
	return answer, data, nil
}
