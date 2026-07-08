// Package ratelimit is a small in-process token-bucket limiter for throttling
// abuse-prone endpoints (OTP, ingest, the public widget). It is per-instance —
// good enough as a first line of defense; a Redis-backed limiter can replace it
// behind the same Middleware when the API runs on more than one node.
package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter hands out tokens per key at a fixed refill rate with a burst ceiling.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64 // max tokens (and the initial allowance)
}

// New builds a limiter allowing `burst` requests immediately, then `perMinute`
// sustained. A background sweeper evicts idle keys so the map can't grow forever.
func New(perMinute, burst int) *Limiter {
	l := &Limiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
	}
	go l.sweep()
	return l
}

// allow reports whether the key may proceed, consuming one token if so.
func (l *Limiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	} else {
		b.tokens += now.Sub(b.last).Seconds() * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) sweep() {
	for range time.Tick(5 * time.Minute) {
		cutoff := time.Now().Add(-10 * time.Minute)
		l.mu.Lock()
		for k, b := range l.buckets {
			if b.last.Before(cutoff) {
				delete(l.buckets, k)
			}
		}
		l.mu.Unlock()
	}
}

// Middleware throttles requests keyed by client IP, returning 429 when the
// bucket is empty.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(ClientIP(r)) {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClientIP is the caller's address, trusting the first X-Forwarded-For hop set
// by the reverse proxy in front of the API, then falling back to RemoteAddr.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(first)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
