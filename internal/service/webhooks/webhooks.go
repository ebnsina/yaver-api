// Package webhooks delivers outbound merchant webhooks. A dispatcher loop drains
// the transactional outbox into deliveries, then POSTs each with an HMAC
// signature, retrying with backoff and dead-lettering after maxAttempts.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/crypto"
	"github.com/ebnsina/yaver-api/pkg/id"
)

const maxAttempts = 5

// backoff[attempts] — retry delay after the Nth failed attempt.
var backoff = []time.Duration{2 * time.Second, 5 * time.Second, 15 * time.Second, 60 * time.Second, 5 * time.Minute}

type Service struct {
	repo   domain.WebhookRepo
	cipher *crypto.Cipher
	http   *http.Client
	log    *slog.Logger
}

func New(repo domain.WebhookRepo, cipher *crypto.Cipher, log *slog.Logger) *Service {
	return &Service{repo: repo, cipher: cipher, http: &http.Client{Timeout: 10 * time.Second}, log: log}
}

// SetEndpoint stores the merchant's delivery URL + a freshly generated signing
// secret (encrypted at rest) and returns the secret once for them to save.
func (s *Service) SetEndpoint(ctx context.Context, orgID domain.OrgID, url string, events []string) (secret string, err error) {
	secret = "whsec_" + randToken()
	enc, err := s.cipher.Encrypt([]byte(secret))
	if err != nil {
		return "", err
	}
	if err := s.repo.UpsertEndpoint(ctx, id.New("whep"), string(orgID), url, enc, events); err != nil {
		return "", err
	}
	return secret, nil
}

// Run drains the outbox and delivers due webhooks on a ticker until ctx is done.
func (s *Service) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Service) tick(ctx context.Context) {
	if _, err := s.repo.DrainOutbox(ctx, 100, func() string { return id.New("whd") }); err != nil {
		s.log.Error("webhook drain", "err", err)
	}
	due, err := s.repo.DueDeliveries(ctx, 50)
	if err != nil {
		s.log.Error("webhook due", "err", err)
		return
	}
	for _, d := range due {
		s.deliver(ctx, d)
	}
}

func (s *Service) deliver(ctx context.Context, d domain.DueDelivery) {
	ep, found, err := s.repo.GetEndpoint(ctx, d.OrgID)
	if err != nil || !found {
		s.fail(ctx, d, 0, "endpoint missing")
		return
	}
	secret, err := s.cipher.Decrypt(ep.SecretEnc)
	if err != nil {
		s.fail(ctx, d, 0, "secret decrypt")
		return
	}

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(ts + "." + string(d.Payload)))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, d.URL, bytes.NewReader(d.Payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Yaver-Event", d.Event)
	req.Header.Set("X-Yaver-Timestamp", ts)
	req.Header.Set("X-Yaver-Signature", sig)
	req.Header.Set("X-Yaver-Delivery-Id", d.ID)

	resp, err := s.http.Do(req)
	if err != nil {
		s.fail(ctx, d, 0, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = s.repo.MarkDelivered(ctx, d.ID, resp.StatusCode)
		return
	}
	s.fail(ctx, d, resp.StatusCode, "non-2xx")
}

// fail reschedules with backoff, or dead-letters after maxAttempts.
func (s *Service) fail(ctx context.Context, d domain.DueDelivery, code int, msg string) {
	attempts := d.Attempts + 1
	if attempts >= maxAttempts {
		_ = s.repo.Reschedule(ctx, d.ID, code, msg, "dead", time.Now())
		s.log.Error("webhook dead-lettered", "id", d.ID, "event", d.Event, "err", msg)
		return
	}
	next := time.Now().Add(backoff[min(d.Attempts, len(backoff)-1)])
	_ = s.repo.Reschedule(ctx, d.ID, code, msg, "pending", next)
}

func randToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
