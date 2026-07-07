// Package local is an in-process Orchestrator: a bounded worker pool that runs
// jobs on background goroutines. It implements domain.Orchestrator so Phase 0
// runs the place_call path with no external engine.
//
// The Hatchet adapter (durable, fairness-keyed, retried) implements the same
// interface and replaces this in production — no change above the port.
package local

import (
	"context"
	"log/slog"
	"sync"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// PlaceCallFunc executes a place_call job. main injects the calls-service method.
type PlaceCallFunc func(ctx context.Context, in domain.PlaceCallInput) error

type Orchestrator struct {
	log       *slog.Logger
	placeCall PlaceCallFunc
	jobs      chan func()
	wg        sync.WaitGroup
}

func New(log *slog.Logger, workers int, placeCall PlaceCallFunc) *Orchestrator {
	o := &Orchestrator{
		log:       log,
		placeCall: placeCall,
		jobs:      make(chan func(), 256),
	}
	for range workers {
		o.wg.Add(1)
		go o.worker()
	}
	return o
}

func (o *Orchestrator) worker() {
	defer o.wg.Done()
	for job := range o.jobs {
		job()
	}
}

func (o *Orchestrator) EnqueuePlaceCall(_ context.Context, in domain.PlaceCallInput) error {
	o.jobs <- func() {
		// Detached context: the job outlives the HTTP request that enqueued it.
		if err := o.placeCall(context.Background(), in); err != nil {
			o.log.Error("place_call failed", "org", in.OrgID, "err", err)
		}
	}
	return nil
}

// Shutdown drains in-flight jobs.
func (o *Orchestrator) Shutdown() {
	close(o.jobs)
	o.wg.Wait()
}
