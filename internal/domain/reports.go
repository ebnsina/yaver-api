package domain

import "context"

// Reporter answers a merchant's natural-language question over their own
// metrics. Provider-agnostic (the "AI custom reports" headline feature): the
// built-in template reporter today, an LLM adapter later — same seam as
// ChatModel / Summarizer. The service supplies the data so the adapter never
// touches the database.
type Reporter interface {
	Answer(ctx context.Context, question string, data AnalyticsOverview) (string, error)
}
