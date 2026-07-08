package ingest

import (
	"context"
	"testing"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type fakeEvents struct{ inserted bool }

func (f *fakeEvents) Insert(context.Context, domain.IngestEvent) (bool, error) {
	return f.inserted, nil
}

type fakeCustomers struct{ dnd bool }

func (f *fakeCustomers) Upsert(context.Context, domain.OrgID, string, string, string) (string, bool, error) {
	return "cust_1", f.dnd, nil
}
func (f *fakeCustomers) ListByOrg(context.Context, domain.OrgID, int) ([]domain.Customer, error) {
	return nil, nil
}
func (f *fakeCustomers) SetDND(context.Context, domain.OrgID, string, bool) error { return nil }

// fakeFlows reports a flow as active only for the names it was told about.
type fakeFlows struct{ active map[string]domain.FlowID }

func (f *fakeFlows) GetActiveFlow(_ context.Context, _ domain.OrgID, name string) (domain.Flow, bool, error) {
	if id, ok := f.active[name]; ok {
		return domain.Flow{ID: id, Name: name}, true, nil
	}
	return domain.Flow{}, false, nil
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

type fakeOrch struct {
	calls []domain.PlaceCallInput
}

func (f *fakeOrch) EnqueuePlaceCall(_ context.Context, in domain.PlaceCallInput) error {
	f.calls = append(f.calls, in)
	return nil
}

func newSvc(dnd bool, active map[string]domain.FlowID) (*Service, *fakeOrch) {
	orch := &fakeOrch{}
	svc := New(&fakeEvents{inserted: true}, &fakeCustomers{dnd: dnd}, &fakeFlows{active: active}, orch)
	return svc, orch
}

func TestRoutesEventTypeToFlow(t *testing.T) {
	active := map[string]domain.FlowID{
		"order_confirm":     "flow_oc",
		"cart_recovery":     "flow_cr",
		"delivery_reminder": "flow_dr",
	}
	cases := []struct {
		eventType string
		wantFlow  domain.FlowID
	}{
		{"order_placed", "flow_oc"},
		{"abandoned_cart", "flow_cr"},
		{"out_for_delivery", "flow_dr"},
	}
	for _, tc := range cases {
		t.Run(tc.eventType, func(t *testing.T) {
			svc, orch := newSvc(false, active)
			_, dup, err := svc.Accept(context.Background(), "org_1", Event{Type: tc.eventType, Phone: "01711223344"})
			if err != nil || dup {
				t.Fatalf("Accept: err=%v dup=%v", err, dup)
			}
			if len(orch.calls) != 1 {
				t.Fatalf("expected 1 enqueued call, got %d", len(orch.calls))
			}
			if orch.calls[0].FlowID != tc.wantFlow {
				t.Errorf("routed to flow %q, want %q", orch.calls[0].FlowID, tc.wantFlow)
			}
		})
	}
}

func TestNoCallWhenFlowMissing(t *testing.T) {
	svc, orch := newSvc(false, map[string]domain.FlowID{}) // no flows built yet
	if _, _, err := svc.Accept(context.Background(), "org_1", Event{Type: "abandoned_cart", Phone: "01711223344"}); err != nil {
		t.Fatal(err)
	}
	if len(orch.calls) != 0 {
		t.Errorf("should not call when no flow exists, got %d", len(orch.calls))
	}
}

func TestNoCallOnDND(t *testing.T) {
	svc, orch := newSvc(true, map[string]domain.FlowID{"order_confirm": "flow_oc"})
	if _, _, err := svc.Accept(context.Background(), "org_1", Event{Type: "order_placed", Phone: "01711223344"}); err != nil {
		t.Fatal(err)
	}
	if len(orch.calls) != 0 {
		t.Errorf("should not call a DND customer, got %d", len(orch.calls))
	}
}

func TestUnmappedEventStoredNoCall(t *testing.T) {
	svc, orch := newSvc(false, map[string]domain.FlowID{"order_confirm": "flow_oc"})
	if _, _, err := svc.Accept(context.Background(), "org_1", Event{Type: "order_cancelled", Phone: "01711223344"}); err != nil {
		t.Fatal(err)
	}
	if len(orch.calls) != 0 {
		t.Errorf("unmapped event should not trigger a call, got %d", len(orch.calls))
	}
}
