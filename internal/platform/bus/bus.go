// Package bus is an in-process, org-scoped pub/sub fan-out for live activity
// events. It backs the dashboard SSE feed. Because it satisfies the domain
// ActivityPublisher/ActivitySubscriber ports, a Postgres LISTEN/NOTIFY adapter
// can replace it behind the same seam once the app runs on more than one process.
package bus

import (
	"context"
	"sync"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type subscriber struct {
	ch chan domain.ActivityEvent
}

// Bus fans published events out to the current subscribers of each org.
type Bus struct {
	mu   sync.RWMutex
	subs map[domain.OrgID]map[*subscriber]struct{}
}

// New returns an empty Bus ready to use.
func New() *Bus {
	return &Bus{subs: make(map[domain.OrgID]map[*subscriber]struct{})}
}

// PublishActivity delivers e to every current subscriber of e.OrgID. Delivery is
// non-blocking: a subscriber whose buffer is full drops the event rather than
// stalling the publisher — the feed is best-effort, not a durable log.
func (b *Bus) PublishActivity(_ context.Context, e domain.ActivityEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for s := range b.subs[e.OrgID] {
		select {
		case s.ch <- e:
		default:
		}
	}
}

// SubscribeActivity registers a subscriber for orgID and returns its event
// channel plus an unsubscribe func. The channel is buffered and closed by the
// unsubscribe func, which is idempotent.
func (b *Bus) SubscribeActivity(orgID domain.OrgID) (<-chan domain.ActivityEvent, func()) {
	s := &subscriber{ch: make(chan domain.ActivityEvent, 16)}

	b.mu.Lock()
	if b.subs[orgID] == nil {
		b.subs[orgID] = make(map[*subscriber]struct{})
	}
	b.subs[orgID][s] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			// Remove under the write lock first so no in-flight PublishActivity
			// (which holds only the read lock) can still be sending on s.ch; only
			// then is it safe to close the channel.
			b.mu.Lock()
			if m := b.subs[orgID]; m != nil {
				delete(m, s)
				if len(m) == 0 {
					delete(b.subs, orgID)
				}
			}
			b.mu.Unlock()
			close(s.ch)
		})
	}
	return s.ch, cancel
}
