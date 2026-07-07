package domain

import (
	"context"
	"time"
)

// ChannelConnection links an org to a Meta channel (WhatsApp / Messenger).
type ChannelConnection struct {
	Type        string // whatsapp | messenger
	ExternalID  string // phone_number_id (WhatsApp) or page_id (Messenger)
	CreatedAt   time.Time
	AccessToken string // decrypted; only populated on the inbound path
	VerifyToken string
}

// ChannelRepo persists channel connections. Access tokens are stored encrypted;
// the repo takes/returns ciphertext (the service holds the cipher).
type ChannelRepo interface {
	Upsert(ctx context.Context, orgID OrgID, typ, externalID string, encToken []byte, verifyToken string) error
	List(ctx context.Context, orgID OrgID) ([]ChannelConnection, error)
	Delete(ctx context.Context, orgID OrgID, typ string) error
	// ByExternalID resolves an inbound message's destination to its org + creds.
	ByExternalID(ctx context.Context, externalID string) (orgID OrgID, typ string, encToken []byte, verifyToken string, found bool, err error)
	// OrgForVerifyToken supports the Meta webhook GET challenge.
	OrgForVerifyToken(ctx context.Context, verifyToken string) (found bool, err error)
}

// OutboundMessage is a reply to deliver on a channel.
type OutboundMessage struct {
	Type           string // whatsapp | messenger
	FromExternalID string // our phone_number_id / page_id
	AccessToken    string
	To             string // recipient's channel user id
	Text           string
}

// MessagingSender delivers a message on a channel. Provider-agnostic seam:
// swap the log sender (testable) for the Meta Graph sender.
type MessagingSender interface {
	Send(ctx context.Context, m OutboundMessage) error
}
