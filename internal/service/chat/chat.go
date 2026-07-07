// Package chat runs the AI chat channel: conversations, messages, and replies.
package chat

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
)

// defaultPrompt steers the assistant when an org hasn't set custom instructions.
const defaultPrompt = "You are a friendly customer-support assistant for a Bangladeshi online store. Be concise and helpful."

type Service struct {
	repo       domain.ChatRepo
	settings   domain.ChatSettingsRepo
	model      domain.ChatModel
	summarizer domain.Summarizer
	insights   domain.InsightRepo
}

func New(repo domain.ChatRepo, settings domain.ChatSettingsRepo, model domain.ChatModel, summarizer domain.Summarizer, insights domain.InsightRepo) *Service {
	return &Service{repo: repo, settings: settings, model: model, summarizer: summarizer, insights: insights}
}

// Settings returns the org's chat/widget settings.
func (s *Service) Settings(ctx context.Context, orgID domain.OrgID) (domain.ChatSettings, error) {
	return s.settings.Get(ctx, orgID)
}

// SaveSettings updates the org's chat/widget settings.
func (s *Service) SaveSettings(ctx context.Context, orgID domain.OrgID, cs domain.ChatSettings) error {
	return s.settings.Upsert(ctx, orgID, cs)
}

// Send appends a user message (starting a conversation if convID is empty),
// generates an assistant reply, stores it, and returns the conversation id and
// the reply text.
func (s *Service) Send(ctx context.Context, orgID domain.OrgID, convID, text string) (string, string, error) {
	if convID == "" {
		convID = id.New("conv")
		if err := s.repo.CreateConversation(ctx, orgID, convID); err != nil {
			return "", "", err
		}
	} else if err := s.owns(ctx, orgID, convID); err != nil {
		return "", "", err
	}

	reply, err := s.reply(ctx, orgID, convID, text)
	if err != nil {
		return "", "", err
	}
	return convID, reply, nil
}

// SendChannel handles an inbound message from a messaging channel (WhatsApp /
// Messenger), threaded by the channel user, and returns the assistant reply.
func (s *Service) SendChannel(ctx context.Context, orgID domain.OrgID, channel, externalUser, text string) (string, error) {
	convID, err := s.repo.FindOrCreateChannelConversation(ctx, orgID, channel, externalUser)
	if err != nil {
		return "", err
	}
	return s.reply(ctx, orgID, convID, text)
}

// reply appends the user's message, generates + stores the assistant reply.
func (s *Service) reply(ctx context.Context, orgID domain.OrgID, convID, text string) (string, error) {
	if err := s.repo.AddMessage(ctx, convID, "user", text); err != nil {
		return "", err
	}
	history, err := s.repo.Messages(ctx, convID)
	if err != nil {
		return "", err
	}
	prompt := defaultPrompt
	if cs, err := s.settings.Get(ctx, orgID); err == nil && cs.Instructions != "" {
		prompt = cs.Instructions
	}
	reply, err := s.model.Reply(ctx, prompt, history)
	if err != nil {
		return "", err
	}
	if err := s.repo.AddMessage(ctx, convID, "assistant", reply); err != nil {
		return "", err
	}
	return reply, nil
}

func (s *Service) List(ctx context.Context, orgID domain.OrgID) ([]domain.Conversation, error) {
	return s.repo.ListConversations(ctx, orgID, 100)
}

// Messages returns a conversation's turns, scoped to the org.
func (s *Service) Messages(ctx context.Context, orgID domain.OrgID, convID string) ([]domain.Message, error) {
	if err := s.owns(ctx, orgID, convID); err != nil {
		return nil, err
	}
	return s.repo.Messages(ctx, convID)
}

// Summarize (re)generates the AI insight for a conversation and caches it. The
// merchant triggers this from the inbox; re-running overwrites the prior insight.
func (s *Service) Summarize(ctx context.Context, orgID domain.OrgID, convID string) (domain.ConversationInsight, error) {
	if err := s.owns(ctx, orgID, convID); err != nil {
		return domain.ConversationInsight{}, err
	}
	history, err := s.repo.Messages(ctx, convID)
	if err != nil {
		return domain.ConversationInsight{}, err
	}
	in, err := s.summarizer.Summarize(ctx, history)
	if err != nil {
		return domain.ConversationInsight{}, err
	}
	if err := s.insights.Save(ctx, orgID, convID, in); err != nil {
		return domain.ConversationInsight{}, err
	}
	// Re-read so the response carries the DB-authoritative created_at.
	saved, _, err := s.insights.Get(ctx, convID)
	if err != nil {
		return domain.ConversationInsight{}, err
	}
	return saved, nil
}

// Insight returns the cached insight for a conversation, or found=false if none
// has been generated yet.
func (s *Service) Insight(ctx context.Context, orgID domain.OrgID, convID string) (domain.ConversationInsight, bool, error) {
	if err := s.owns(ctx, orgID, convID); err != nil {
		return domain.ConversationInsight{}, false, err
	}
	return s.insights.Get(ctx, convID)
}

func (s *Service) owns(ctx context.Context, orgID domain.OrgID, convID string) error {
	owner, _, found, err := s.repo.GetConversation(ctx, convID)
	if err != nil {
		return err
	}
	if !found || owner != orgID {
		return domain.ErrNotFound
	}
	return nil
}
