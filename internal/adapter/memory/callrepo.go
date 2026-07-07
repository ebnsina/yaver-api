// Package memory holds in-memory repo implementations used in Phase 0 / tests.
// These are swapped for the Postgres (sqlc) repos as the DB lands — same
// domain interfaces, so nothing above the repo changes.
package memory

import (
	"context"
	"sync"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type CallRepo struct {
	mu    sync.RWMutex
	calls map[domain.CallID]*domain.Call
}

func NewCallRepo() *CallRepo {
	return &CallRepo{calls: make(map[domain.CallID]*domain.Call)}
}

func (r *CallRepo) Create(_ context.Context, c *domain.Call) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *c
	r.calls[c.ID] = &cp
	return nil
}

func (r *CallRepo) Get(_ context.Context, id domain.CallID) (*domain.Call, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.calls[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}
