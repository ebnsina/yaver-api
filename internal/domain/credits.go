package domain

import (
	"context"
	"time"
)

// CreditEntry is one line in the credit ledger.
type CreditEntry struct {
	Delta     int
	Reason    string
	CreatedAt time.Time
}

// CreditRepo meters usage: a balance per org plus an audit ledger.
type CreditRepo interface {
	Balance(ctx context.Context, orgID OrgID) (int, error)
	// TryDeduct atomically subtracts amount if the balance covers it. ok=false
	// (no charge) when funds are insufficient.
	TryDeduct(ctx context.Context, orgID OrgID, amount int, reason string) (ok bool, balance int, err error)
	// Grant adds credits (e.g., a top-up or signup grant) and returns the balance.
	Grant(ctx context.Context, orgID OrgID, amount int, reason string) (balance int, err error)
	Ledger(ctx context.Context, orgID OrgID, limit int) ([]CreditEntry, error)
}
