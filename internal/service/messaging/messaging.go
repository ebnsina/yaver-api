// Package messaging connects Meta channels (WhatsApp / Messenger) to the chat
// assistant: it stores connections, and routes inbound messages through the
// assistant back out to the sender.
package messaging

import (
	"context"
	"log/slog"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/crypto"
)

// Assistant is the chat surface messaging needs (satisfied by *chat.Service).
type Assistant interface {
	SendChannel(ctx context.Context, orgID domain.OrgID, channel, externalUser, text string) (string, error)
}

type Service struct {
	conns     domain.ChannelRepo
	assistant Assistant
	sender    domain.MessagingSender
	cipher    *crypto.Cipher
	log       *slog.Logger
}

func New(conns domain.ChannelRepo, assistant Assistant, sender domain.MessagingSender, cipher *crypto.Cipher, log *slog.Logger) *Service {
	return &Service{conns: conns, assistant: assistant, sender: sender, cipher: cipher, log: log}
}

// Connect stores (or updates) a channel connection, encrypting the token.
func (s *Service) Connect(ctx context.Context, orgID domain.OrgID, typ, externalID, accessToken, verifyToken string) error {
	enc, err := s.cipher.Encrypt([]byte(accessToken))
	if err != nil {
		return err
	}
	return s.conns.Upsert(ctx, orgID, typ, externalID, enc, verifyToken)
}

func (s *Service) List(ctx context.Context, orgID domain.OrgID) ([]domain.ChannelConnection, error) {
	return s.conns.List(ctx, orgID)
}

func (s *Service) Disconnect(ctx context.Context, orgID domain.OrgID, typ string) error {
	return s.conns.Delete(ctx, orgID, typ)
}

// VerifyChallenge reports whether a Meta webhook verify_token matches a
// connection (used for the GET subscription handshake).
func (s *Service) VerifyChallenge(ctx context.Context, verifyToken string) bool {
	ok, err := s.conns.OrgForVerifyToken(ctx, verifyToken)
	return err == nil && ok
}

// HandleInbound routes one inbound message: resolve the org from the receiving
// channel id, run the assistant, and send the reply back out.
func (s *Service) HandleInbound(ctx context.Context, toExternalID, from, text string) error {
	orgID, typ, encToken, _, found, err := s.conns.ByExternalID(ctx, toExternalID)
	if err != nil {
		return err
	}
	if !found {
		s.log.Warn("inbound for unknown channel", "external_id", toExternalID)
		return nil
	}
	reply, err := s.assistant.SendChannel(ctx, orgID, typ, from, text)
	if err != nil {
		return err
	}
	token, err := s.cipher.Decrypt(encToken)
	if err != nil {
		return err
	}
	return s.sender.Send(ctx, domain.OutboundMessage{
		Type: typ, FromExternalID: toExternalID, AccessToken: string(token), To: from, Text: reply,
	})
}
