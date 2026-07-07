package builtin

import (
	"context"
	"testing"

	"github.com/ebnsina/yaver-api/internal/domain"
)

func msgs(pairs ...[2]string) []domain.Message {
	out := make([]domain.Message, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, domain.Message{Role: p[0], Content: p[1]})
	}
	return out
}

func TestSummarizeOutcomes(t *testing.T) {
	s := NewSummarizer()
	cases := []struct {
		name      string
		history   []domain.Message
		outcome   string
		sentiment string
	}{
		{
			name:      "empty",
			history:   nil,
			outcome:   domain.OutcomeUnknown,
			sentiment: domain.SentimentNeutral,
		},
		{
			name:      "human request wins over sale words",
			history:   msgs([2]string{"user", "I want to order but let me talk to a human agent"}),
			outcome:   domain.OutcomeNeedsHuman,
			sentiment: domain.SentimentNeutral,
		},
		{
			name:      "buying intent",
			history:   msgs([2]string{"user", "I want to buy the blue shirt"}),
			outcome:   domain.OutcomeSale,
			sentiment: domain.SentimentNeutral,
		},
		{
			name:      "resolved and positive",
			history:   msgs([2]string{"user", "great, thanks!"}),
			outcome:   domain.OutcomeResolved,
			sentiment: domain.SentimentPositive,
		},
		{
			name:      "negative escalates to human",
			history:   msgs([2]string{"user", "this is a scam, terrible service"}),
			outcome:   domain.OutcomeNeedsHuman,
			sentiment: domain.SentimentNegative,
		},
		{
			name:      "plain question pends",
			history:   msgs([2]string{"user", "where is my parcel"}),
			outcome:   domain.OutcomePending,
			sentiment: domain.SentimentNeutral,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in, err := s.Summarize(context.Background(), tc.history)
			if err != nil {
				t.Fatalf("Summarize: %v", err)
			}
			if in.Outcome != tc.outcome {
				t.Errorf("outcome = %q, want %q", in.Outcome, tc.outcome)
			}
			if in.Sentiment != tc.sentiment {
				t.Errorf("sentiment = %q, want %q", in.Sentiment, tc.sentiment)
			}
			if in.Summary == "" || in.NextAction == "" {
				t.Errorf("summary/next_action must be non-empty, got %+v", in)
			}
		})
	}
}

func TestSummarizeTruncatesLongMessage(t *testing.T) {
	long := ""
	for range 200 {
		long += "x"
	}
	in, err := NewSummarizer().Summarize(context.Background(), msgs([2]string{"user", long}))
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(in.Summary)) > 200 {
		t.Errorf("summary should be truncated, len=%d", len([]rune(in.Summary)))
	}
}
