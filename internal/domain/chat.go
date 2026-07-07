package domain

import (
	"context"
	"time"
)

// Conversation is a chat thread between a customer and the assistant.
type Conversation struct {
	ID           string
	Channel      string
	Status       string // open | closed
	LastMessage  string
	MessageCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Message is one turn in a conversation.
type Message struct {
	Role      string // user | assistant | system
	Content   string
	CreatedAt time.Time
}

// ChatModel generates an assistant reply. This is the provider-agnostic seam:
// swap the adapter (built-in, OpenAI, Anthropic, …) without touching the service.
type ChatModel interface {
	// Reply returns the assistant's next message given a system prompt and the
	// prior turns (oldest first).
	Reply(ctx context.Context, system string, history []Message) (string, error)
}

// ChatRepo persists conversations and messages.
type ChatRepo interface {
	CreateConversation(ctx context.Context, orgID OrgID, id string) error
	// GetConversation returns the conversation's owning org (caller checks it).
	GetConversation(ctx context.Context, id string) (orgID OrgID, status string, found bool, err error)
	ListConversations(ctx context.Context, orgID OrgID, limit int) ([]Conversation, error)
	AddMessage(ctx context.Context, conversationID, role, content string) error
	Messages(ctx context.Context, conversationID string) ([]Message, error)
}
