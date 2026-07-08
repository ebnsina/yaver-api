package billing

import (
	"context"
	"testing"

	paymentmock "github.com/ebnsina/yaver-api/internal/adapter/payment/mock"
	"github.com/ebnsina/yaver-api/internal/domain"
)

type fakeCredits struct{ granted int }

func (f *fakeCredits) Balance(context.Context, domain.OrgID) (int, error) { return f.granted, nil }
func (f *fakeCredits) TryDeduct(context.Context, domain.OrgID, int, string) (bool, int, error) {
	return true, 0, nil
}
func (f *fakeCredits) Grant(_ context.Context, _ domain.OrgID, amount int, _ string) (int, error) {
	f.granted += amount
	return f.granted, nil
}
func (f *fakeCredits) Ledger(context.Context, domain.OrgID, int) ([]domain.CreditEntry, error) {
	return nil, nil
}

// fakePayments mimics the postgres Settle transition semantics in memory.
type fakePayments struct{ byRef map[string]*domain.Payment }

func newFakePayments() *fakePayments { return &fakePayments{byRef: map[string]*domain.Payment{}} }

func (f *fakePayments) Create(_ context.Context, p *domain.Payment) error {
	cp := *p
	f.byRef[p.ProviderRef] = &cp
	return nil
}
func (f *fakePayments) Settle(_ context.Context, _, providerRef string, status domain.PaymentStatus) (domain.Payment, bool, error) {
	p, ok := f.byRef[providerRef]
	if !ok {
		return domain.Payment{}, false, domain.ErrNotFound
	}
	if p.Status != domain.PaymentPending {
		return *p, false, nil // already settled
	}
	p.Status = status
	return *p, true, nil
}

func newSvc() (*Service, *fakeCredits, *fakePayments) {
	credits := &fakeCredits{}
	payments := newFakePayments()
	return New(credits, payments, paymentmock.New("http://localhost:8080")), credits, payments
}

func TestCheckoutReturnsRedirectAndPendsPayment(t *testing.T) {
	svc, credits, payments := newSvc()
	url, err := svc.Checkout(context.Background(), "org_1", 500)
	if err != nil {
		t.Fatal(err)
	}
	if url == "" {
		t.Fatal("expected a redirect url")
	}
	if len(payments.byRef) != 1 {
		t.Fatalf("expected 1 pending payment, got %d", len(payments.byRef))
	}
	if credits.granted != 0 {
		t.Fatal("credits must not be granted before payment confirms")
	}
}

func TestIPNGrantsCreditsExactlyOnce(t *testing.T) {
	svc, credits, payments := newSvc()
	if _, err := svc.Checkout(context.Background(), "org_1", 500); err != nil {
		t.Fatal(err)
	}
	var ref string
	for r := range payments.byRef {
		ref = r
	}

	form := map[string]string{"tran_id": ref, "status": "paid"}
	// Replayed IPN (at-least-once delivery) must not double-grant.
	for range 3 {
		if err := svc.SettlePayment(context.Background(), form); err != nil {
			t.Fatal(err)
		}
	}
	if credits.granted != 500 {
		t.Fatalf("expected 500 credits granted exactly once, got %d", credits.granted)
	}
}

func TestIPNFailedDoesNotGrant(t *testing.T) {
	svc, credits, payments := newSvc()
	if _, err := svc.Checkout(context.Background(), "org_1", 500); err != nil {
		t.Fatal(err)
	}
	var ref string
	for r := range payments.byRef {
		ref = r
	}
	if err := svc.SettlePayment(context.Background(), map[string]string{"tran_id": ref, "status": "failed"}); err != nil {
		t.Fatal(err)
	}
	if credits.granted != 0 {
		t.Fatalf("failed payment must not grant credits, got %d", credits.granted)
	}
}

func TestIPNUnknownRefIgnored(t *testing.T) {
	svc, credits, _ := newSvc()
	if err := svc.SettlePayment(context.Background(), map[string]string{"tran_id": "nope", "status": "paid"}); err != nil {
		t.Fatalf("unknown ref should be ignored, got %v", err)
	}
	if credits.granted != 0 {
		t.Fatal("no grant for unknown ref")
	}
}
