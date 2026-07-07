// Package builtin is a dependency-free ChatModel: a small rule-based assistant.
// It exists so the chat channel works end-to-end without an external LLM. Swap
// it for an OpenAI/Anthropic adapter (same domain.ChatModel interface) later.
package builtin

import (
	"context"
	"strings"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type Model struct{}

func New() *Model { return &Model{} }

// intent maps a set of trigger words to a canned reply.
type intent struct {
	triggers []string
	reply    string
}

var intents = []intent{
	{[]string{"hi", "hello", "hey", "salam", "assalam"},
		"Hello! 👋 I'm here to help with your order — tracking, returns, delivery, or anything else. What do you need?"},
	{[]string{"track", "where", "status", "order status", "kobe", "delivery date"},
		"I can check that for you. Could you share your order number? Most orders arrive within 2–3 working days inside Dhaka and 3–5 days outside."},
	{[]string{"return", "refund", "exchange", "ferot"},
		"No problem — returns are accepted within 7 days of delivery for unused items. Would you like me to start a return request?"},
	{[]string{"delivery", "shipping", "charge", "cost of delivery", "koto"},
		"Delivery is 60৳ inside Dhaka and 120৳ outside. Cash on delivery is available everywhere."},
	{[]string{"cancel"},
		"I can help cancel that. If the order hasn't shipped yet, cancellation is free — please share your order number."},
	{[]string{"thanks", "thank", "dhonnobad"},
		"You're welcome! 🙏 Anything else I can help with?"},
}

func (m *Model) Reply(_ context.Context, _ string, history []domain.Message) (string, error) {
	last := ""
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			last = strings.ToLower(history[i].Content)
			break
		}
	}
	if last == "" {
		return "Hello! How can I help you today?", nil
	}
	for _, in := range intents {
		for _, t := range in.triggers {
			if strings.Contains(last, t) {
				return in.reply, nil
			}
		}
	}
	return "Thanks for your message! I've noted it and a team member will follow up shortly. Meanwhile, I can help with order tracking, returns, or delivery info.", nil
}
