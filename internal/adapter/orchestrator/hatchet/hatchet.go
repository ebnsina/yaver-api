// Package hatchet implements domain.Orchestrator on Hatchet (self-hosted).
//
// place_call is a standalone task with a per-merchant fairness key
// (concurrency expression "input.orgId", GROUP_ROUND_ROBIN) so one merchant
// can't monopolize call slots. The SDK reads its connection token from the
// environment (HATCHET_CLIENT_TOKEN; HATCHET_CLIENT_TLS_STRATEGY=none for
// insecure self-host). Same domain.Orchestrator interface as the local
// dispatcher — swapping engines is a wiring change, nothing above the port.
package hatchet

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/sdks/go"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// PlaceCallFunc executes the place_call job (the calls-service method).
type PlaceCallFunc func(context.Context, domain.PlaceCallInput) error

type placeCallInput struct {
	OrgID   string `json:"orgId"`
	ToPhone string `json:"toPhone"`
	FlowID  string `json:"flowId"`
}

type placeCallResult struct {
	OK bool `json:"ok"`
}

type Orchestrator struct {
	client    *v1.Client
	placeCall *v1.StandaloneTask
}

func New(handler PlaceCallFunc) (*Orchestrator, error) {
	client, err := v1.NewClient()
	if err != nil {
		return nil, err
	}

	maxRuns := int32(50)
	strategy := types.GroupRoundRobin
	task := client.NewStandaloneTask("place_call",
		func(ctx v1.Context, in placeCallInput) (placeCallResult, error) {
			err := handler(ctx, domain.PlaceCallInput{
				OrgID:   domain.OrgID(in.OrgID),
				ToPhone: in.ToPhone,
				FlowID:  domain.FlowID(in.FlowID),
			})
			return placeCallResult{OK: err == nil}, err
		},
		v1.WithConcurrency(&types.Concurrency{
			Expression:    "input.orgId", // per-merchant fairness key
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategy,
		}),
	)

	return &Orchestrator{client: client, placeCall: task}, nil
}

// EnqueuePlaceCall triggers the durable task (fire-and-forget).
func (o *Orchestrator) EnqueuePlaceCall(ctx context.Context, in domain.PlaceCallInput) error {
	_, err := o.placeCall.RunNoWait(ctx, placeCallInput{
		OrgID:   string(in.OrgID),
		ToPhone: in.ToPhone,
		FlowID:  string(in.FlowID),
	})
	return err
}

// StartWorker runs the worker loop (blocking). Call it in a goroutine.
func (o *Orchestrator) StartWorker(ctx context.Context) error {
	w, err := o.client.NewWorker("yaver-worker", v1.WithWorkflows(o.placeCall))
	if err != nil {
		return err
	}
	return w.StartBlocking(ctx)
}
