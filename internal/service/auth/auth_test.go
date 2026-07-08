package auth

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// fakeRepo is an in-memory domain.AuthRepo for testing the service logic.
type fakeRepo struct {
	users    map[string]domain.User
	otps     map[string]*storedOTP // keyed by phone (latest)
	sessions map[string]string     // tokenHashHex -> userID
	seq      int
}

type storedOTP struct {
	id       string
	hash     []byte
	attempts int
	expires  time.Time
	consumed bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: map[string]domain.User{}, otps: map[string]*storedOTP{}, sessions: map[string]string{}}
}

func (r *fakeRepo) UpsertUserByPhone(_ context.Context, phone string) (domain.User, error) {
	if u, ok := r.users[phone]; ok {
		return u, nil
	}
	r.seq++
	u := domain.User{ID: "user_" + hex.EncodeToString([]byte{byte(r.seq)}), Phone: phone}
	r.users[phone] = u
	return u, nil
}

func (r *fakeRepo) InsertOTP(_ context.Context, phone string, hash []byte, exp time.Time) error {
	r.seq++
	r.otps[phone] = &storedOTP{id: "otp_" + hex.EncodeToString([]byte{byte(r.seq)}), hash: hash, expires: exp}
	return nil
}

func (r *fakeRepo) LatestLiveOTP(_ context.Context, phone string) (domain.OTP, bool, error) {
	o, ok := r.otps[phone]
	if !ok || o.consumed {
		return domain.OTP{}, false, nil
	}
	return domain.OTP{ID: o.id, CodeHash: o.hash, Attempts: o.attempts}, true, nil
}

func (r *fakeRepo) IncrementOTPAttempts(_ context.Context, otpID string) error {
	for _, o := range r.otps {
		if o.id == otpID {
			o.attempts++
		}
	}
	return nil
}

func (r *fakeRepo) ConsumeOTP(_ context.Context, otpID string) error {
	for _, o := range r.otps {
		if o.id == otpID {
			o.consumed = true
		}
	}
	return nil
}

func (r *fakeRepo) CreateSession(_ context.Context, tokenHash []byte, userID string, _ time.Time) error {
	r.sessions[hex.EncodeToString(tokenHash)] = userID
	return nil
}

func (r *fakeRepo) SessionUser(_ context.Context, tokenHash []byte) (domain.SessionUser, bool, error) {
	uid, ok := r.sessions[hex.EncodeToString(tokenHash)]
	if !ok {
		return domain.SessionUser{}, false, nil
	}
	return domain.SessionUser{UserID: uid}, true, nil
}

func (r *fakeRepo) DeleteSession(_ context.Context, tokenHash []byte) error {
	delete(r.sessions, hex.EncodeToString(tokenHash))
	return nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type fakeSMS struct{ sent int }

func (f *fakeSMS) Send(context.Context, string, string) error { f.sent++; return nil }

func TestOTPLoginFlow(t *testing.T) {
	repo := newFakeRepo()
	sms := &fakeSMS{}
	svc := New(repo, fixedClock{time.Unix(1_700_000_000, 0)}, "test-secret", "dev", sms)
	ctx := context.Background()
	const phone = "+8801712345678"

	code, err := svc.RequestOTP(ctx, phone)
	if sms.sent != 1 {
		t.Fatalf("RequestOTP should send exactly one SMS, sent %d", sms.sent)
	}
	if err != nil || code == "" {
		t.Fatalf("request otp: code=%q err=%v", code, err)
	}

	// Wrong code is rejected and counts an attempt.
	if _, _, err := svc.VerifyOTP(ctx, phone, "000000"); err != domain.ErrInvalidOTP {
		t.Fatalf("wrong code: want ErrInvalidOTP, got %v", err)
	}

	// Correct code issues a session.
	token, su, err := svc.VerifyOTP(ctx, phone, code)
	if err != nil || token == "" {
		t.Fatalf("verify: token=%q err=%v", token, err)
	}
	if su.Phone != phone {
		t.Fatalf("session user phone = %q, want %q", su.Phone, phone)
	}

	// The session resolves back to the same user.
	got, found, err := svc.Resolve(ctx, token)
	if err != nil || !found || got.UserID != su.UserID {
		t.Fatalf("resolve: found=%v user=%q err=%v", found, got.UserID, err)
	}

	// The code is single-use (already consumed).
	if _, _, err := svc.VerifyOTP(ctx, phone, code); err != domain.ErrInvalidOTP {
		t.Fatalf("reuse: want ErrInvalidOTP, got %v", err)
	}

	// Logout revokes the session.
	if err := svc.Logout(ctx, token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, found, _ := svc.Resolve(ctx, token); found {
		t.Fatalf("session should be revoked after logout")
	}
}
