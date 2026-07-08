// Package onboarding is the AI setup assistant: it answers a merchant's
// setup questions with the configured (provider-agnostic) chat model, grounded
// in Yaver's setup steps and the org's current progress. Stateless — it doesn't
// persist as a customer conversation.
package onboarding

import (
	"context"
	"fmt"
	"strings"

	"github.com/ebnsina/yaver-api/internal/domain"
)

const systemPrompt = `You are Yaver's setup assistant. Yaver is a multi-channel customer-engagement platform for e-commerce — voice IVR, an embeddable AI chat widget, and WhatsApp/Messenger — that turns store events into confirmed orders, recovered carts, and answered questions.

Help the merchant finish setup. The core steps are:
1. Name your store (Settings).
2. Send a test call from the Dashboard.
3. Create a secret API key and POST store events to /v1/events.
4. Connect a webhook to receive outcomes.
Then: embed the chat widget with a publishable key, or connect WhatsApp/Messenger.

Be concise, friendly, and concrete. Prefer a next action over a long explanation. If asked something unrelated to using Yaver, gently steer back.`

// Progress is the org's current setup state, folded into the prompt so answers
// reference what's left to do.
type Progress struct {
	StoreNamed bool
	HasAPIKey  bool
	WebhookSet bool
	HasFlow    bool
}

type Service struct {
	model domain.ChatModel
}

func New(model domain.ChatModel) *Service { return &Service{model: model} }

// Ask answers one onboarding question given the org's progress.
func (s *Service) Ask(ctx context.Context, p Progress, question string) (string, error) {
	system := systemPrompt + "\n\nCurrent setup status:\n" + statusLines(p)
	return s.model.Reply(ctx, system, []domain.Message{{Role: "user", Content: question}})
}

func statusLines(p Progress) string {
	mark := func(done bool) string {
		if done {
			return "done"
		}
		return "not done"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "- Store named: %s\n", mark(p.StoreNamed))
	fmt.Fprintf(&b, "- API key created: %s\n", mark(p.HasAPIKey))
	fmt.Fprintf(&b, "- Webhook connected: %s\n", mark(p.WebhookSet))
	fmt.Fprintf(&b, "- Has a flow: %s", mark(p.HasFlow))
	return b.String()
}
