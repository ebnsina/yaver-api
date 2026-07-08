package calls

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// orderConfirmFlow is a minimal valid IVR the engine can drive to a terminal.
func orderConfirmFlow() domain.Flow {
	return domain.Flow{
		ID:     "flow_1",
		Type:   domain.FlowIVR,
		Locale: "bn",
		IVR: domain.IVRSpec{
			Entry: "greet",
			Nodes: map[string]domain.IVRNode{
				"greet": {
					Say:    &domain.Prompt{TTS: "confirm your order"},
					Gather: &domain.Gather{Digits: 1, TimeoutS: 6},
					On:     map[string]string{"1": "confirmed", "2": "cancelled", "timeout": "no_input"},
				},
				"confirmed": {Result: "confirmed", End: true},
				"cancelled": {Result: "cancelled", End: true},
				"no_input":  {Result: "no_answer", End: true},
			},
		},
	}
}

type fakeFlows struct {
	flow  domain.Flow
	found bool
	err   error
}

func (f *fakeFlows) GetActiveFlow(context.Context, domain.OrgID, string) (domain.Flow, bool, error) {
	return f.flow, f.found, f.err
}
func (f *fakeFlows) Create(context.Context, domain.OrgID, domain.NewFlow) (domain.FlowID, error) {
	return "", nil
}
func (f *fakeFlows) ListByOrg(context.Context, domain.OrgID) ([]domain.FlowSummary, error) {
	return nil, nil
}
func (f *fakeFlows) GetByID(context.Context, domain.FlowID) (domain.FlowDetail, bool, error) {
	return domain.FlowDetail{}, false, nil
}
func (f *fakeFlows) UpdateSpec(context.Context, domain.FlowID, domain.OrgID, []byte) error {
	return nil
}

type fakeCredits struct {
	ok      bool
	balance int
	err     error
	charged int
}

func (c *fakeCredits) Balance(context.Context, domain.OrgID) (int, error) { return c.balance, nil }
func (c *fakeCredits) TryDeduct(_ context.Context, _ domain.OrgID, amount int, _ string) (bool, int, error) {
	if c.err != nil {
		return false, 0, c.err
	}
	if c.ok {
		c.charged += amount
	}
	return c.ok, c.balance, nil
}
func (c *fakeCredits) Grant(context.Context, domain.OrgID, int, string) (int, error) {
	return c.balance, nil
}
func (c *fakeCredits) Ledger(context.Context, domain.OrgID, int) ([]domain.CreditEntry, error) {
	return nil, nil
}

type fakeProvider struct {
	pid    domain.ProviderCallID
	err    error
	called int
}

func (p *fakeProvider) PlaceCall(context.Context, domain.CallRequest) (domain.ProviderCallID, error) {
	p.called++
	return p.pid, p.err
}

type fakeOutcomes struct {
	recorded *domain.Call
	err      error
}

func (o *fakeOutcomes) RecordCallOutcome(_ context.Context, c *domain.Call, _ *domain.OutboxEvent) error {
	o.recorded = c
	return o.err
}

type fakeCalls struct{}

func (fakeCalls) Create(context.Context, *domain.Call) error { return nil }
func (fakeCalls) Get(context.Context, domain.CallID) (*domain.Call, error) {
	return nil, domain.ErrNotFound
}
func (fakeCalls) ListByOrg(context.Context, domain.OrgID, int) ([]domain.Call, error) {
	return nil, nil
}
func (fakeCalls) Summary(context.Context, domain.OrgID) (domain.CallSummary, error) {
	return domain.CallSummary{}, nil
}

type fakePolicy struct{}

func (fakePolicy) Get(context.Context, domain.OrgID) (domain.CallPolicy, error) {
	return domain.CallPolicy{}, nil
}
func (fakePolicy) Upsert(context.Context, domain.OrgID, domain.CallPolicy) error { return nil }

type fakeNotifier struct{ calls int }

func (n *fakeNotifier) LowBalance(context.Context, domain.OrgID, int) { n.calls++ }

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Unix(1700000000, 0) }

type fakeActivity struct{ events []domain.ActivityEvent }

func (a *fakeActivity) PublishActivity(_ context.Context, e domain.ActivityEvent) {
	a.events = append(a.events, e)
}

func newSvc(flows *fakeFlows, credits *fakeCredits, provider *fakeProvider, outcomes *fakeOutcomes, act *fakeActivity) *Service {
	return New(provider, outcomes, fakeCalls{}, flows, credits, fakePolicy{}, &fakeNotifier{}, fakeClock{}, act)
}

func TestRunTestCall_NoActiveFlow(t *testing.T) {
	credits := &fakeCredits{ok: true}
	provider := &fakeProvider{}
	svc := newSvc(&fakeFlows{found: false}, credits, provider, &fakeOutcomes{}, &fakeActivity{})

	_, _, err := svc.RunTestCall(context.Background(), "org_1", "01712345678", "1", "order_confirm")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if credits.charged != 0 {
		t.Fatal("must not charge when there's no flow")
	}
	if provider.called != 0 {
		t.Fatal("must not dial when there's no flow")
	}
}

func TestRunTestCall_InsufficientCredits(t *testing.T) {
	credits := &fakeCredits{ok: false} // deduction fails
	provider := &fakeProvider{}
	svc := newSvc(&fakeFlows{flow: orderConfirmFlow(), found: true}, credits, provider, &fakeOutcomes{}, &fakeActivity{})

	_, _, err := svc.RunTestCall(context.Background(), "org_1", "01712345678", "1", "order_confirm")
	if !errors.Is(err, domain.ErrInsufficientCredits) {
		t.Fatalf("want ErrInsufficientCredits, got %v", err)
	}
	if provider.called != 0 {
		t.Fatal("must not dial when out of credits")
	}
}

func TestRunTestCall_HappyPath(t *testing.T) {
	credits := &fakeCredits{ok: true}
	provider := &fakeProvider{pid: "pcall_1"}
	outcomes := &fakeOutcomes{}
	act := &fakeActivity{}
	svc := newSvc(&fakeFlows{flow: orderConfirmFlow(), found: true}, credits, provider, outcomes, act)

	out, _, err := svc.RunTestCall(context.Background(), "org_1", "01712345678", "1", "order_confirm")
	if err != nil {
		t.Fatal(err)
	}
	if out.Result != "confirmed" {
		t.Fatalf("digit 1 should confirm, got %q", out.Result)
	}
	if credits.charged != creditsPerCall {
		t.Fatalf("should charge %d, got %d", creditsPerCall, credits.charged)
	}
	if provider.called != 1 {
		t.Fatal("should place exactly one call")
	}
	if outcomes.recorded == nil || outcomes.recorded.Result != "confirmed" {
		t.Fatal("should persist the confirmed call")
	}
	if outcomes.recorded.OrgID != "org_1" || outcomes.recorded.ProviderCallID != "pcall_1" {
		t.Fatal("recorded call should carry org + provider id")
	}
	if len(act.events) != 1 || act.events[0].Type != "call."+string(out.Status) || act.events[0].Detail != "confirmed" {
		t.Fatalf("should publish one activity event for the confirmed call, got %v", act.events)
	}
}

func TestRunTestCall_DialFailureDoesNotRecord(t *testing.T) {
	credits := &fakeCredits{ok: true}
	provider := &fakeProvider{err: errors.New("carrier down")}
	outcomes := &fakeOutcomes{}
	svc := newSvc(&fakeFlows{flow: orderConfirmFlow(), found: true}, credits, provider, outcomes, &fakeActivity{})

	if _, _, err := svc.RunTestCall(context.Background(), "org_1", "01712345678", "1", "order_confirm"); err == nil {
		t.Fatal("expected a dial error")
	}
	if outcomes.recorded != nil {
		t.Fatal("must not record a call the provider never placed")
	}
}
