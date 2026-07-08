package chat

import (
	"context"
	"testing"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// fakeChatRepo is an in-memory ChatRepo for behavior tests.
type fakeChatRepo struct {
	status   string
	channel  string
	extUser  string
	messages []domain.Message
}

func (f *fakeChatRepo) CreateConversation(context.Context, domain.OrgID, string) error { return nil }
func (f *fakeChatRepo) GetConversation(context.Context, string) (domain.OrgID, string, bool, error) {
	return "org_1", f.status, true, nil
}
func (f *fakeChatRepo) ChannelTarget(context.Context, string) (string, string, error) {
	return f.channel, f.extUser, nil
}
func (f *fakeChatRepo) SetStatus(_ context.Context, _, status string) error {
	f.status = status
	return nil
}
func (f *fakeChatRepo) ListConversations(context.Context, domain.OrgID, int) ([]domain.Conversation, error) {
	return nil, nil
}
func (f *fakeChatRepo) AddMessage(_ context.Context, _, role, content string) error {
	f.messages = append(f.messages, domain.Message{Role: role, Content: content})
	return nil
}
func (f *fakeChatRepo) Messages(context.Context, string) ([]domain.Message, error) {
	return f.messages, nil
}
func (f *fakeChatRepo) FindOrCreateChannelConversation(context.Context, domain.OrgID, string, string) (string, error) {
	return "conv_1", nil
}

type fakeSettings struct{}

func (fakeSettings) Get(context.Context, domain.OrgID) (domain.ChatSettings, error) {
	return domain.DefaultChatSettings(), nil
}
func (fakeSettings) Upsert(context.Context, domain.OrgID, domain.ChatSettings) error { return nil }

type fakeModel struct{ calls int }

func (m *fakeModel) Reply(context.Context, string, []domain.Message) (string, error) {
	m.calls++
	return "bot reply", nil
}

func newSvc(repo *fakeChatRepo, model *fakeModel) *Service {
	return New(repo, fakeSettings{}, model, nil, nil, nil)
}

func TestAssistantRepliesWhenOpen(t *testing.T) {
	repo := &fakeChatRepo{status: domain.ConvOpen}
	model := &fakeModel{}
	_, reply, err := newSvc(repo, model).Send(context.Background(), "org_1", "conv_1", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "bot reply" || model.calls != 1 {
		t.Errorf("expected assistant reply, got reply=%q calls=%d", reply, model.calls)
	}
}

func TestAssistantSilentWhenHandling(t *testing.T) {
	repo := &fakeChatRepo{status: domain.ConvHandling}
	model := &fakeModel{}
	_, reply, err := newSvc(repo, model).Send(context.Background(), "org_1", "conv_1", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "" {
		t.Errorf("assistant should stay silent during human takeover, got %q", reply)
	}
	if model.calls != 0 {
		t.Errorf("model must not be called while handling, calls=%d", model.calls)
	}
	// The customer's message is still recorded for the human to read.
	if len(repo.messages) != 1 || repo.messages[0].Role != "user" {
		t.Errorf("customer message should be stored, got %+v", repo.messages)
	}
}

func TestAgentReplyMarksHandlingAndStores(t *testing.T) {
	repo := &fakeChatRepo{status: domain.ConvOpen, channel: "whatsapp", extUser: "wa_123"}
	ch, user, err := newSvc(repo, &fakeModel{}).AgentReply(context.Background(), "org_1", "conv_1", "let me help")
	if err != nil {
		t.Fatal(err)
	}
	if ch != "whatsapp" || user != "wa_123" {
		t.Errorf("expected channel target, got %q/%q", ch, user)
	}
	if repo.status != domain.ConvHandling {
		t.Errorf("agent reply should mark conversation handling, got %q", repo.status)
	}
	if len(repo.messages) != 1 || repo.messages[0].Role != "agent" {
		t.Errorf("agent message should be stored with role agent, got %+v", repo.messages)
	}
}

func TestSetStatusRejectsUnknown(t *testing.T) {
	if err := newSvc(&fakeChatRepo{status: domain.ConvOpen}, &fakeModel{}).
		SetStatus(context.Background(), "org_1", "conv_1", "bogus"); err != domain.ErrFlowInvalid {
		t.Errorf("expected ErrFlowInvalid for bad status, got %v", err)
	}
}
