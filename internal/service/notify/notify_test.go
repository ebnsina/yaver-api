package notify

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type fakeEmail struct {
	sent int
	to   string
	err  error
}

func (f *fakeEmail) Send(_ context.Context, to, _, _ string) error {
	f.sent++
	f.to = to
	return f.err
}

type fakeOrgs struct {
	email string
	err   error
}

func (f *fakeOrgs) EnsureForUser(context.Context, string, string) (domain.Org, error) {
	return domain.Org{}, nil
}
func (f *fakeOrgs) Rename(context.Context, domain.OrgID, string) error { return nil }
func (f *fakeOrgs) OwnerEmail(context.Context, domain.OrgID) (string, error) {
	return f.email, f.err
}

func newSvc(email domain.EmailSender, orgs domain.OrgStore) *Service {
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), email, orgs)
}

func TestLowBalanceSendsWhenEmailPresent(t *testing.T) {
	em := &fakeEmail{}
	newSvc(em, &fakeOrgs{email: "owner@shop.test"}).LowBalance(context.Background(), "org_1", 5)
	if em.sent != 1 {
		t.Fatalf("expected 1 email, got %d", em.sent)
	}
	if em.to != "owner@shop.test" {
		t.Errorf("sent to %q", em.to)
	}
}

func TestLowBalanceSkipsWhenNoEmail(t *testing.T) {
	em := &fakeEmail{}
	newSvc(em, &fakeOrgs{email: ""}).LowBalance(context.Background(), "org_1", 5)
	if em.sent != 0 {
		t.Errorf("should not send when owner has no email, sent %d", em.sent)
	}
}

func TestLowBalanceSwallowsErrors(t *testing.T) {
	// Neither a lookup error nor a send error should panic or propagate.
	newSvc(&fakeEmail{}, &fakeOrgs{err: errors.New("db down")}).LowBalance(context.Background(), "o", 1)
	newSvc(&fakeEmail{err: errors.New("smtp down")}, &fakeOrgs{email: "x@y.z"}).LowBalance(context.Background(), "o", 1)
}
