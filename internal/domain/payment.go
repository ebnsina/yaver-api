package domain

import (
	"context"
	"time"
)

// PaymentStatus is the lifecycle of a top-up attempt.
type PaymentStatus string

const (
	PaymentPending PaymentStatus = "pending"
	PaymentPaid    PaymentStatus = "paid"
	PaymentFailed  PaymentStatus = "failed"
)

// Payment is one credit top-up attempt: a pending row created at checkout,
// settled to paid/failed when the gateway confirms via IPN.
type Payment struct {
	ID          string
	OrgID       OrgID
	Provider    string // "mock" | "sslcommerz"
	ProviderRef string // gateway transaction id (we use the payment id as tran_id)
	Credits     int
	AmountBDT   int
	Status      PaymentStatus
	CreatedAt   time.Time
}

// PaymentRepo persists top-up attempts and settles them idempotently.
type PaymentRepo interface {
	Create(ctx context.Context, p *Payment) error
	// Settle marks a pending payment paid/failed by (provider, ref) and reports
	// whether this call performed the transition (false if it was already
	// settled). At-least-once IPN delivery means callers must grant credits only
	// when transitioned is true.
	Settle(ctx context.Context, provider, providerRef string, status PaymentStatus) (p Payment, transitioned bool, err error)
}

// PaymentRequest starts a hosted checkout.
type PaymentRequest struct {
	OrgID     OrgID
	PaymentID string // our payment id — echoed back by the gateway as tran_id
	Credits   int
	AmountBDT int
}

// PaymentCheckout is the gateway's response: where to send the customer to pay.
type PaymentCheckout struct {
	ProviderRef string
	RedirectURL string
}

// PaymentResult is the verified outcome parsed from a gateway IPN callback.
type PaymentResult struct {
	ProviderRef string
	Status      PaymentStatus
}

// PaymentGateway abstracts a payment aggregator (SSLCommerz — which fronts
// bKash/Nagad/cards — or the dev mock). Another provider is a new adapter.
type PaymentGateway interface {
	Name() string
	// Checkout starts a hosted payment session and returns a redirect URL.
	Checkout(ctx context.Context, req PaymentRequest) (PaymentCheckout, error)
	// VerifyIPN authenticates and parses a gateway callback (its POST form
	// values). ok is false when the callback fails authentication and must be
	// ignored — never grant credits on an unverified callback.
	VerifyIPN(ctx context.Context, form map[string]string) (result PaymentResult, ok bool, err error)
}
