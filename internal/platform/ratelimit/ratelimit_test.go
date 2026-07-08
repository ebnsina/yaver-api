package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBurstThenThrottle(t *testing.T) {
	l := New(60, 3) // burst 3

	// The burst allowance passes...
	for i := range 3 {
		if !l.allow("1.2.3.4") {
			t.Fatalf("request %d within burst should pass", i)
		}
	}
	// ...the next is throttled (refill at 1/sec can't have accrued instantly).
	if l.allow("1.2.3.4") {
		t.Fatal("request past the burst should be throttled")
	}
	// A different key has its own bucket.
	if !l.allow("5.6.7.8") {
		t.Fatal("a different IP must not be affected")
	}
}

func TestMiddlewareReturns429(t *testing.T) {
	l := New(60, 1)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func() int {
		r := httptest.NewRequest(http.MethodPost, "/x", nil)
		r.RemoteAddr = "9.9.9.9:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if got := call(); got != http.StatusOK {
		t.Fatalf("first call: got %d", got)
	}
	if got := call(); got != http.StatusTooManyRequests {
		t.Fatalf("second call: want 429, got %d", got)
	}
}

func TestClientIPPrefersForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:5000"
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	if ip := ClientIP(r); ip != "203.0.113.9" {
		t.Fatalf("want first XFF hop, got %q", ip)
	}
}
