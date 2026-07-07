package builtin

import (
	"context"
	"strings"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// Summarizer is a dependency-free domain.Summarizer: it derives a conversation
// insight from keyword signals in the transcript. Like Model, it exists so the
// enrichment feature works end-to-end without an external LLM — swap it for an
// OpenAI/Anthropic adapter (same interface) to get real summaries.
type Summarizer struct{}

func NewSummarizer() *Summarizer { return &Summarizer{} }

// signal words that push the outcome/sentiment. Kept small and BD-store-flavored
// to match the built-in Model's intents (English + a few common Bangla words).
var (
	negativeWords = []string{"angry", "worst", "terrible", "scam", "cheat", "refund", "complaint", "kharap", "baje", "problem"}
	positiveWords = []string{"thanks", "thank", "great", "perfect", "dhonnobad", "good", "love", "happy"}
	humanWords    = []string{"human", "agent", "manager", "speak to", "talk to someone", "representative", "call me"}
	saleWords     = []string{"buy", "order", "purchase", "checkout", "confirm", "place order", "kinbo", "kinte chai"}
	resolvedWords = []string{"thanks", "thank", "solved", "got it", "understood", "dhonnobad", "ok thanks"}
)

func (s *Summarizer) Summarize(_ context.Context, history []domain.Message) (domain.ConversationInsight, error) {
	var userText, lastUser string
	userTurns := 0
	for _, m := range history {
		if m.Role != "user" {
			continue
		}
		userTurns++
		lastUser = m.Content
		userText += " " + strings.ToLower(m.Content)
	}

	if userTurns == 0 {
		return domain.ConversationInsight{
			Summary:    "No customer messages yet.",
			Outcome:    domain.OutcomeUnknown,
			Sentiment:  domain.SentimentNeutral,
			NextAction: "Wait for the customer to reply.",
		}, nil
	}

	sentiment := domain.SentimentNeutral
	if containsAny(userText, positiveWords) {
		sentiment = domain.SentimentPositive
	}
	if containsAny(userText, negativeWords) {
		sentiment = domain.SentimentNegative // negative overrides a stray "thanks"
	}

	// Outcome, most-actionable first: a human ask trumps everything.
	outcome := domain.OutcomePending
	nextAction := "Follow up with the customer to close the loop."
	switch {
	case containsAny(userText, humanWords):
		outcome, nextAction = domain.OutcomeNeedsHuman, "Assign a team member — the customer asked for a person."
	case containsAny(userText, saleWords):
		outcome, nextAction = domain.OutcomeSale, "Confirm the order details and payment method."
	case containsAny(userText, resolvedWords):
		outcome, nextAction = domain.OutcomeResolved, "No action needed — the customer's question was answered."
	case sentiment == domain.SentimentNegative:
		outcome, nextAction = domain.OutcomeNeedsHuman, "Reach out personally — the customer sounds unhappy."
	}

	return domain.ConversationInsight{
		Summary:    summarize(lastUser, userTurns),
		Outcome:    outcome,
		Sentiment:  sentiment,
		NextAction: nextAction,
	}, nil
}

// summarize builds a short recap from the customer's last message.
func summarize(lastUser string, turns int) string {
	msg := strings.TrimSpace(lastUser)
	const max = 140
	if len(msg) > max {
		msg = strings.TrimSpace(msg[:max]) + "…"
	}
	turnWord := "message"
	if turns != 1 {
		turnWord = "messages"
	}
	return itoa(turns) + " customer " + turnWord + "; latest: \"" + msg + "\""
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// itoa avoids pulling strconv for a tiny count.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
