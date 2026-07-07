package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/ebnsina/yaver-api/internal/domain"
)

func sampleData() domain.AnalyticsOverview {
	return domain.AnalyticsOverview{
		Calls:         domain.CallSummary{Total: 10, Confirmed: 6, Cancelled: 2, Today: 3},
		Conversations: domain.ConversationStats{Total: 8, Today: 4, Resolved: 5, Pending: 1, Sale: 2, NeedsHuman: 2},
		Credits:       domain.CreditSummary{Balance: 400, SpentToday: 12, Spent30d: 100},
	}
}

func TestReporterRoutesByTopic(t *testing.T) {
	r := New()
	d := sampleData()
	cases := []struct {
		question string
		wants    []string // substrings the answer must contain
	}{
		{"how many calls did we confirm?", []string{"10 calls", "6 confirmed", "60%"}},
		{"what's my chat resolution rate?", []string{"8 conversations", "5 resolved", "62%"}},
		{"how many sales did the assistant find?", []string{"2 conversations with buying intent"}},
		{"how much have I spent?", []string{"balance is 400", "12 credits today", "100"}},
		{"give me a summary", []string{"10 calls", "8 conversations", "balance: 400"}},
	}
	for _, tc := range cases {
		t.Run(tc.question, func(t *testing.T) {
			ans, err := r.Answer(context.Background(), tc.question, d)
			if err != nil {
				t.Fatal(err)
			}
			low := strings.ToLower(ans)
			for _, w := range tc.wants {
				if !strings.Contains(low, strings.ToLower(w)) {
					t.Errorf("answer %q missing %q", ans, w)
				}
			}
		})
	}
}

func TestReporterHandlesZeroTotals(t *testing.T) {
	// No data should not divide by zero; rates read 0.
	ans, err := New().Answer(context.Background(), "confirmation rate?", domain.AnalyticsOverview{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ans, "0%") {
		t.Errorf("expected a 0%% rate, got %q", ans)
	}
}
