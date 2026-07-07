// Package builtin is a dependency-free domain.Reporter: it answers a merchant's
// question from their metrics using keyword routing + templated sentences. It
// exists so AI custom reports work end-to-end without an external LLM — swap it
// for an OpenAI/Anthropic adapter (same interface) for free-form answers.
package builtin

import (
	"context"
	"strconv"
	"strings"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type Reporter struct{}

func New() *Reporter { return &Reporter{} }

// topic groups the question into a metric family we can answer from the numbers.
type topic int

const (
	topicOverview topic = iota
	topicCalls
	topicChat
	topicSales
	topicSpend
)

func classify(q string) topic {
	switch {
	case containsAny(q, "sale", "sold", "upsell", "purchase", "revenue"):
		return topicSales
	case containsAny(q, "spend", "spent", "cost", "credit", "balance", "budget", "money"):
		return topicSpend
	case containsAny(q, "call", "confirm", "recover", "cancel", "ivr", "voice"):
		return topicCalls
	case containsAny(q, "chat", "conversation", "message", "resolve", "resolved", "customer", "whatsapp", "messenger"):
		return topicChat
	default:
		return topicOverview
	}
}

func (r *Reporter) Answer(_ context.Context, question string, d domain.AnalyticsOverview) (string, error) {
	q := strings.ToLower(strings.TrimSpace(question))
	switch classify(q) {
	case topicCalls:
		return callsAnswer(d), nil
	case topicChat:
		return chatAnswer(d), nil
	case topicSales:
		return "AI chat flagged " + n(d.Conversations.Sale) + " conversations with buying intent" +
			" out of " + n(d.Conversations.Total) + " total. " + needsHumanTail(d) + focusTail(), nil
	case topicSpend:
		return "Your credit balance is " + n(d.Credits.Balance) + ". You've spent " +
			n(d.Credits.SpentToday) + " credits today and " + n(d.Credits.Spent30d) +
			" in the last 30 days.", nil
	default:
		return overviewAnswer(d), nil
	}
}

func callsAnswer(d domain.AnalyticsOverview) string {
	c := d.Calls
	return n(c.Total) + " calls placed (" + n(c.Today) + " today), " + n(c.Confirmed) +
		" confirmed and " + n(c.Cancelled) + " cancelled — a " + pct(c.Confirmed, c.Total) +
		"% confirmation rate."
}

func chatAnswer(d domain.AnalyticsOverview) string {
	c := d.Conversations
	return n(c.Total) + " conversations (" + n(c.Today) + " today): " + n(c.Resolved) +
		" resolved, " + n(c.Pending) + " pending, " + n(c.NeedsHuman) +
		" need a human — a " + pct(c.Resolved, c.Total) + "% resolution rate. " + needsHumanTail(d)
}

func overviewAnswer(d domain.AnalyticsOverview) string {
	return callsAnswer(d) + " " + chatAnswer(d) + " Credit balance: " + n(d.Credits.Balance) + "."
}

// needsHumanTail nudges the merchant when conversations are waiting on a person.
func needsHumanTail(d domain.AnalyticsOverview) string {
	if d.Conversations.NeedsHuman > 0 {
		return n(d.Conversations.NeedsHuman) + " conversation(s) are waiting on a team member."
	}
	return ""
}

func focusTail() string {
	return " Follow up with the high-intent ones first."
}

func containsAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func n(i int) string { return strconv.Itoa(i) }

func pct(part, total int) string {
	if total <= 0 {
		return "0"
	}
	return strconv.Itoa(int(float64(part) / float64(total) * 100))
}
