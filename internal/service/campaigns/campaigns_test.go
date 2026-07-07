package campaigns

import (
	"context"
	"testing"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// --- fakes ---

type fakeRepo struct {
	campaigns  map[string]*domain.Campaign
	orgs       map[string]domain.OrgID
	recipients map[string][]domain.Recipient
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		campaigns:  map[string]*domain.Campaign{},
		orgs:       map[string]domain.OrgID{},
		recipients: map[string][]domain.Recipient{},
	}
}
func (r *fakeRepo) put(id string, org domain.OrgID, status string, sched *time.Time) {
	r.campaigns[id] = &domain.Campaign{ID: id, Status: status, ScheduledAt: sched}
	r.orgs[id] = org
}
func (r *fakeRepo) Create(_ context.Context, org domain.OrgID, id, name string) error {
	r.put(id, org, "draft", nil)
	return nil
}
func (r *fakeRepo) ListByOrg(context.Context, domain.OrgID) ([]domain.Campaign, error) {
	return nil, nil
}
func (r *fakeRepo) Get(_ context.Context, id string) (domain.Campaign, domain.OrgID, bool, error) {
	c, ok := r.campaigns[id]
	if !ok {
		return domain.Campaign{}, "", false, nil
	}
	return *c, r.orgs[id], true, nil
}
func (r *fakeRepo) MarkStarted(_ context.Context, _ domain.OrgID, id string, count int) error {
	r.campaigns[id].Status = "completed"
	r.campaigns[id].TargetCount = count
	return nil
}
func (r *fakeRepo) AddRecipient(_ context.Context, _ domain.OrgID, id string, rec domain.Recipient) error {
	for _, existing := range r.recipients[id] {
		if existing.Phone == rec.Phone {
			return nil // dedup
		}
	}
	r.recipients[id] = append(r.recipients[id], rec)
	return nil
}
func (r *fakeRepo) Recipients(_ context.Context, id string) ([]domain.Recipient, error) {
	return r.recipients[id], nil
}
func (r *fakeRepo) Schedule(_ context.Context, _ domain.OrgID, id string, at time.Time) error {
	r.campaigns[id].Status = "scheduled"
	r.campaigns[id].ScheduledAt = &at
	return nil
}
func (r *fakeRepo) DueCampaigns(_ context.Context, now time.Time) ([]domain.CampaignRef, error) {
	var out []domain.CampaignRef
	for id, c := range r.campaigns {
		if c.Status == "scheduled" && c.ScheduledAt != nil && !c.ScheduledAt.After(now) {
			out = append(out, domain.CampaignRef{ID: id, OrgID: r.orgs[id]})
		}
	}
	return out, nil
}

type fakeCustomers struct {
	dnd  map[string]bool   // phone -> dnd
	list []domain.Customer // fallback list
	ups  []string          // phones upserted
}

func (f *fakeCustomers) Upsert(_ context.Context, _ domain.OrgID, phone, _, _ string) (string, bool, error) {
	f.ups = append(f.ups, phone)
	return "cust", f.dnd[phone], nil
}
func (f *fakeCustomers) ListByOrg(context.Context, domain.OrgID, int) ([]domain.Customer, error) {
	return f.list, nil
}
func (f *fakeCustomers) SetDND(context.Context, domain.OrgID, string, bool) error { return nil }

type fakeFlows struct{}

func (fakeFlows) GetActiveFlow(context.Context, domain.OrgID, string) (domain.Flow, bool, error) {
	return domain.Flow{ID: "flow_oc"}, true, nil
}
func (fakeFlows) Create(context.Context, domain.OrgID, domain.NewFlow) (domain.FlowID, error) {
	return "", nil
}
func (fakeFlows) ListByOrg(context.Context, domain.OrgID) ([]domain.FlowSummary, error) {
	return nil, nil
}
func (fakeFlows) GetByID(context.Context, domain.FlowID) (domain.FlowDetail, bool, error) {
	return domain.FlowDetail{}, false, nil
}
func (fakeFlows) UpdateSpec(context.Context, domain.FlowID, domain.OrgID, []byte) error { return nil }

type fakeOrch struct{ calls []domain.PlaceCallInput }

func (f *fakeOrch) EnqueuePlaceCall(_ context.Context, in domain.PlaceCallInput) error {
	f.calls = append(f.calls, in)
	return nil
}

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

func newSvc(repo *fakeRepo, cust *fakeCustomers, now time.Time) (*Service, *fakeOrch) {
	orch := &fakeOrch{}
	return New(repo, cust, fakeFlows{}, orch, fakeClock{t: now}), orch
}

// --- tests ---

func TestImportRecipientsParsesAndDedups(t *testing.T) {
	repo := newFakeRepo()
	repo.put("camp_1", "org_1", "draft", nil)
	svc, _ := newSvc(repo, &fakeCustomers{}, time.Unix(1000, 0))

	csv := "phone,name\n01711223344,Rahim\n01711223344,Dup\n0181-234-5678,Karim\ngarbage\n"
	added, err := svc.ImportRecipients(context.Background(), "org_1", "camp_1", csv)
	if err != nil {
		t.Fatal(err)
	}
	if added != 2 { // header + dup + garbage all skipped
		t.Errorf("added = %d, want 2", added)
	}
	if got := len(repo.recipients["camp_1"]); got != 2 {
		t.Errorf("stored %d recipients, want 2", got)
	}
}

func TestStartDispatchesToRecipientsSkippingDND(t *testing.T) {
	repo := newFakeRepo()
	repo.put("camp_1", "org_1", "draft", nil)
	repo.recipients["camp_1"] = []domain.Recipient{
		{Phone: "+8801711111111"}, {Phone: "+8801722222222"},
	}
	cust := &fakeCustomers{dnd: map[string]bool{"+8801722222222": true}}
	svc, orch := newSvc(repo, cust, time.Unix(1000, 0))

	queued, err := svc.Start(context.Background(), "org_1", "camp_1")
	if err != nil {
		t.Fatal(err)
	}
	if queued != 1 || len(orch.calls) != 1 {
		t.Fatalf("expected 1 call (DND skipped), got queued=%d calls=%d", queued, len(orch.calls))
	}
	if orch.calls[0].ToPhone != "+8801711111111" || orch.calls[0].FlowID != "flow_oc" {
		t.Errorf("wrong dispatch: %+v", orch.calls[0])
	}
	if repo.campaigns["camp_1"].Status != "completed" {
		t.Errorf("campaign should be completed after start")
	}
}

func TestStartFallsBackToAllCustomers(t *testing.T) {
	repo := newFakeRepo()
	repo.put("camp_1", "org_1", "draft", nil) // no recipients imported
	cust := &fakeCustomers{list: []domain.Customer{
		{Phone: "+8801711111111"}, {Phone: "+8801722222222", DND: true},
	}}
	svc, orch := newSvc(repo, cust, time.Unix(1000, 0))

	queued, err := svc.Start(context.Background(), "org_1", "camp_1")
	if err != nil {
		t.Fatal(err)
	}
	if queued != 1 || len(orch.calls) != 1 {
		t.Errorf("expected 1 fallback call, got queued=%d calls=%d", queued, len(orch.calls))
	}
}

func TestScheduleRejectsPast(t *testing.T) {
	repo := newFakeRepo()
	repo.put("camp_1", "org_1", "draft", nil)
	now := time.Unix(1000, 0)
	svc, _ := newSvc(repo, &fakeCustomers{}, now)

	if err := svc.Schedule(context.Background(), "org_1", "camp_1", now.Add(-time.Hour)); err != domain.ErrFlowInvalid {
		t.Errorf("past schedule should be rejected, got %v", err)
	}
	if err := svc.Schedule(context.Background(), "org_1", "camp_1", now.Add(time.Hour)); err != nil {
		t.Fatalf("future schedule should succeed: %v", err)
	}
	if repo.campaigns["camp_1"].Status != "scheduled" {
		t.Errorf("campaign should be scheduled")
	}
}

func TestRunDueStartsArrivedCampaigns(t *testing.T) {
	repo := newFakeRepo()
	now := time.Unix(2000, 0)
	past := now.Add(-time.Minute)
	future := now.Add(time.Hour)
	repo.put("due", "org_1", "scheduled", &past)
	repo.put("notyet", "org_1", "scheduled", &future)
	repo.recipients["due"] = []domain.Recipient{{Phone: "+8801711111111"}}
	svc, orch := newSvc(repo, &fakeCustomers{}, now)

	started, err := svc.RunDue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if started != 1 {
		t.Errorf("started = %d, want 1", started)
	}
	if repo.campaigns["due"].Status != "completed" {
		t.Errorf("due campaign should be completed")
	}
	if repo.campaigns["notyet"].Status != "scheduled" {
		t.Errorf("future campaign should stay scheduled")
	}
	if len(orch.calls) != 1 {
		t.Errorf("expected 1 dispatched call, got %d", len(orch.calls))
	}
}
