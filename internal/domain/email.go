package domain

import "context"

// EmailSender delivers a transactional email. Provider-agnostic: a log adapter
// in dev, Resend (or any provider) in prod — swapping is a new adapter.
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// CreditNotifier is told when an org's balance runs low after a charge. Fire-
// and-forget: the call path must not fail because a notification couldn't be
// sent, so there's no error to handle at the call site.
type CreditNotifier interface {
	LowBalance(ctx context.Context, orgID OrgID, balance int)
}
