package bus

import (
	"context"
	"testing"

	"github.com/ebnsina/yaver-api/internal/domain"
)

func TestDeliversToOrgSubscribers(t *testing.T) {
	b := New()
	ch, cancel := b.SubscribeActivity("org_1")
	defer cancel()

	b.PublishActivity(context.Background(), domain.ActivityEvent{OrgID: "org_1", Type: "call.completed"})

	select {
	case e := <-ch:
		if e.Type != "call.completed" {
			t.Fatalf("got type %q", e.Type)
		}
	default:
		t.Fatal("expected an event")
	}
}

func TestIsolatesByOrg(t *testing.T) {
	b := New()
	ch, cancel := b.SubscribeActivity("org_1")
	defer cancel()

	// Event for a different org must not reach org_1's subscriber.
	b.PublishActivity(context.Background(), domain.ActivityEvent{OrgID: "org_2", Type: "chat.message"})

	select {
	case e := <-ch:
		t.Fatalf("unexpected cross-org event: %v", e)
	default:
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	b := New()
	ch, cancel := b.SubscribeActivity("org_1")
	cancel()

	// Channel is closed after cancel; a receive yields the zero value with ok=false.
	if _, ok := <-ch; ok {
		t.Fatal("channel should be closed after cancel")
	}
	// Publishing after unsubscribe must not panic.
	b.PublishActivity(context.Background(), domain.ActivityEvent{OrgID: "org_1", Type: "call.completed"})
}

func TestCancelIsIdempotent(t *testing.T) {
	b := New()
	_, cancel := b.SubscribeActivity("org_1")
	cancel()
	cancel() // second call must be a no-op, not a double-close panic
}

func TestDropsWhenBufferFull(t *testing.T) {
	b := New()
	_, cancel := b.SubscribeActivity("org_1")
	defer cancel()

	// Far more than the buffer (16); a slow consumer must not block the publisher.
	for range 100 {
		b.PublishActivity(context.Background(), domain.ActivityEvent{OrgID: "org_1", Type: "call.completed"})
	}
}
