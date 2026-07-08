// Package anthropic implements domain.ChatModel with Anthropic's Claude models
// via the official SDK. It's one concrete adapter behind the provider-agnostic
// ChatModel seam — swapping to another vendor is a new adapter, not a refactor.
package anthropic

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/ebnsina/yaver-api/internal/domain"
)

// maxReplyTokens caps a single assistant reply. Support-chat turns are short, so
// a modest cap keeps latency and cost down.
const maxReplyTokens = 1024

// Model adapts Claude to the ChatModel port.
type Model struct {
	client anthropic.Client
	model  anthropic.Model
}

var _ domain.ChatModel = (*Model)(nil)

// New returns a Claude-backed ChatModel. model defaults to Claude Opus 4.8 when
// empty. The API key comes from config, never hardcoded.
func New(apiKey, model string) *Model {
	m := anthropic.ModelClaudeOpus4_8
	if model != "" {
		m = anthropic.Model(model)
	}
	return &Model{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  m,
	}
}

// Reply generates the assistant's next message. The system prompt steers tone
// and policy; history holds the prior turns (oldest first). System-role turns in
// history are folded into the top-level system prompt — the Messages API keeps
// system separate from the user/assistant transcript, which must start with a
// user turn (our chat flow always appends the user message before calling this).
func (m *Model) Reply(ctx context.Context, system string, history []domain.Message) (string, error) {
	msgs := make([]anthropic.MessageParam, 0, len(history))
	for _, h := range history {
		block := anthropic.NewTextBlock(h.Content)
		switch h.Role {
		case "assistant":
			msgs = append(msgs, anthropic.NewAssistantMessage(block))
		case "system":
			system = strings.TrimSpace(system + "\n" + h.Content)
		default:
			msgs = append(msgs, anthropic.NewUserMessage(block))
		}
	}

	resp, err := m.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     m.model,
		MaxTokens: maxReplyTokens,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages:  msgs,
	})
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			b.WriteString(t.Text)
		}
	}
	return strings.TrimSpace(b.String()), nil
}
