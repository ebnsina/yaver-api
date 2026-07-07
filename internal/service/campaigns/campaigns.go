// Package campaigns runs batch outbound calls over an imported recipient list
// (or, when none is imported, the org's callable customers). Campaigns can be
// started on demand or scheduled to fire at a future time.
package campaigns

import (
	"context"
	"encoding/csv"
	"io"
	"strings"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
	"github.com/ebnsina/yaver-api/pkg/phone"
)

// campaignFlow is the flow batch calls run (the order-confirmation wedge).
const campaignFlow = "order_confirm"

type Service struct {
	repo      domain.CampaignRepo
	customers domain.CustomerRepo
	flows     domain.FlowRepo
	orch      domain.Orchestrator
	clock     domain.Clock
}

func New(repo domain.CampaignRepo, customers domain.CustomerRepo, flows domain.FlowRepo, orch domain.Orchestrator, clock domain.Clock) *Service {
	return &Service{repo: repo, customers: customers, flows: flows, orch: orch, clock: clock}
}

func (s *Service) Create(ctx context.Context, orgID domain.OrgID, name string) (string, error) {
	cid := id.New("camp")
	if err := s.repo.Create(ctx, orgID, cid, name); err != nil {
		return "", err
	}
	return cid, nil
}

func (s *Service) List(ctx context.Context, orgID domain.OrgID) ([]domain.Campaign, error) {
	return s.repo.ListByOrg(ctx, orgID)
}

func (s *Service) Get(ctx context.Context, orgID domain.OrgID, id string) (domain.Campaign, error) {
	c, owner, found, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	if !found || owner != orgID {
		return domain.Campaign{}, domain.ErrNotFound
	}
	return c, nil
}

// ImportRecipients parses a CSV (one recipient per row: phone[,name]), normalizes
// and dedups phones, and adds them to the campaign. Header rows and rows with an
// invalid phone are skipped. Returns how many valid recipients were imported.
func (s *Service) ImportRecipients(ctx context.Context, orgID domain.OrgID, campaignID, csvData string) (int, error) {
	if _, err := s.Get(ctx, orgID, campaignID); err != nil {
		return 0, err
	}
	reader := csv.NewReader(strings.NewReader(csvData))
	reader.FieldsPerRecord = -1 // rows may be phone or phone,name

	seen := map[string]bool{}
	added := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return added, domain.ErrFlowInvalid // malformed CSV
		}
		if len(row) == 0 {
			continue
		}
		e164, perr := phone.NormalizeBD(strings.TrimSpace(row[0]))
		if perr != nil {
			continue // header or junk row — skip
		}
		if seen[e164] {
			continue // duplicate within this file
		}
		seen[e164] = true
		name := ""
		if len(row) > 1 {
			name = strings.TrimSpace(row[1])
		}
		if err := s.repo.AddRecipient(ctx, orgID, campaignID, domain.Recipient{Phone: e164, Name: name}); err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

// Schedule sets a draft campaign to auto-start at a future time.
func (s *Service) Schedule(ctx context.Context, orgID domain.OrgID, id string, at time.Time) error {
	if !at.After(s.clock.Now()) {
		return domain.ErrFlowInvalid // must be in the future
	}
	if _, err := s.Get(ctx, orgID, id); err != nil {
		return err
	}
	return s.repo.Schedule(ctx, orgID, id, at)
}

// Start dispatches the campaign now and marks it completed. Idempotent: a
// completed campaign is a no-op returning its dispatched count.
func (s *Service) Start(ctx context.Context, orgID domain.OrgID, id string) (int, error) {
	c, err := s.Get(ctx, orgID, id)
	if err != nil {
		return 0, err
	}
	if c.Status == "completed" {
		return c.TargetCount, nil
	}
	queued, err := s.dispatch(ctx, orgID, id)
	if err != nil {
		return queued, err
	}
	if err := s.repo.MarkStarted(ctx, orgID, id, queued); err != nil {
		return queued, err
	}
	return queued, nil
}

// RunDue starts every scheduled campaign whose time has arrived — called on a
// timer by the process. Returns how many campaigns were started. A campaign that
// fails to dispatch is left scheduled and retried on the next sweep.
func (s *Service) RunDue(ctx context.Context) (int, error) {
	due, err := s.repo.DueCampaigns(ctx, s.clock.Now())
	if err != nil {
		return 0, err
	}
	started := 0
	for _, ref := range due {
		queued, derr := s.dispatch(ctx, ref.OrgID, ref.ID)
		if derr != nil {
			continue // leave scheduled; next sweep retries
		}
		if err := s.repo.MarkStarted(ctx, ref.OrgID, ref.ID, queued); err != nil {
			continue
		}
		started++
	}
	return started, nil
}

// dispatch enqueues a call for each non-DND target: the campaign's imported
// recipients, or (when none were imported) all of the org's callable customers.
func (s *Service) dispatch(ctx context.Context, orgID domain.OrgID, campaignID string) (int, error) {
	var flowID domain.FlowID
	if f, found, ferr := s.flows.GetActiveFlow(ctx, orgID, campaignFlow); ferr == nil && found {
		flowID = f.ID
	}

	type target struct {
		phone string
		dnd   bool
	}
	var targets []target

	recips, err := s.repo.Recipients(ctx, campaignID)
	if err != nil {
		return 0, err
	}
	if len(recips) > 0 {
		for _, r := range recips {
			// Upsert reflects the recipient as a customer and returns current DND.
			_, dnd, uerr := s.customers.Upsert(ctx, orgID, r.Phone, r.Name, "")
			if uerr != nil {
				return 0, uerr
			}
			targets = append(targets, target{r.Phone, dnd})
		}
	} else {
		custs, cerr := s.customers.ListByOrg(ctx, orgID, 500)
		if cerr != nil {
			return 0, cerr
		}
		for _, c := range custs {
			targets = append(targets, target{c.Phone, c.DND})
		}
	}

	queued := 0
	for _, t := range targets {
		if t.dnd {
			continue
		}
		if err := s.orch.EnqueuePlaceCall(ctx, domain.PlaceCallInput{OrgID: orgID, ToPhone: t.phone, FlowID: flowID}); err != nil {
			return queued, err
		}
		queued++
	}
	return queued, nil
}
