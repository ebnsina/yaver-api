package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
)

type ChatRepo struct{ q *gen.Queries }

func NewChatRepo(pool *pgxpool.Pool) *ChatRepo { return &ChatRepo{q: gen.New(pool)} }

func (r *ChatRepo) CreateConversation(ctx context.Context, orgID domain.OrgID, cid string) error {
	return r.q.CreateConversation(ctx, gen.CreateConversationParams{ID: cid, OrgID: string(orgID)})
}

func (r *ChatRepo) FindOrCreateChannelConversation(ctx context.Context, orgID domain.OrgID, channel, externalUser string) (string, error) {
	existing, err := r.q.FindOpenChannelConversation(ctx, gen.FindOpenChannelConversationParams{
		OrgID: string(orgID), Channel: channel, ExternalUser: strPtr(externalUser),
	})
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	cid := id.New("conv")
	return cid, r.q.CreateChannelConversation(ctx, gen.CreateChannelConversationParams{
		ID: cid, OrgID: string(orgID), Channel: channel, ExternalUser: strPtr(externalUser),
	})
}

func (r *ChatRepo) GetConversation(ctx context.Context, cid string) (domain.OrgID, string, bool, error) {
	row, err := r.q.GetConversation(ctx, cid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return domain.OrgID(row.OrgID), row.Status, true, nil
}

func (r *ChatRepo) ListConversations(ctx context.Context, orgID domain.OrgID, limit int) ([]domain.Conversation, error) {
	rows, err := r.q.ListConversationsByOrg(ctx, gen.ListConversationsByOrgParams{OrgID: string(orgID), Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Conversation, 0, len(rows))
	for _, row := range rows {
		last, _ := row.LastMessage.(string) // COALESCE'd text
		out = append(out, domain.Conversation{
			ID: row.ID, Status: row.Status, LastMessage: last, MessageCount: int(row.MessageCount),
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *ChatRepo) AddMessage(ctx context.Context, conversationID, role, content string) error {
	if err := r.q.InsertMessage(ctx, gen.InsertMessageParams{
		ID: id.New("msg"), ConversationID: conversationID, Role: role, Content: content,
	}); err != nil {
		return err
	}
	return r.q.TouchConversation(ctx, conversationID)
}

func (r *ChatRepo) Messages(ctx context.Context, conversationID string) ([]domain.Message, error) {
	rows, err := r.q.ListMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Message{Role: row.Role, Content: row.Content, CreatedAt: row.CreatedAt})
	}
	return out, nil
}
