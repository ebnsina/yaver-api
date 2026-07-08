// Package auth implements phone-OTP login with server-side sessions.
//
// OTP codes are stored as HMAC-SHA256(auth_secret, code) with a short TTL and
// an attempt cap. Session tokens are high-entropy random values; only their
// SHA-256 hash is stored, so a DB leak can't be replayed as a cookie.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// minPasswordLen is the shortest password we accept at registration.
const minPasswordLen = 8

const (
	otpTTL      = 5 * time.Minute
	sessionTTL  = 30 * 24 * time.Hour
	maxAttempts = 5
)

type Service struct {
	repo    domain.AuthRepo
	clock   domain.Clock
	secret  []byte
	sms     domain.SMSSender
	devEcho bool // in dev, RequestOTP also returns the code for convenience
}

func New(repo domain.AuthRepo, clock domain.Clock, secret, env string, sms domain.SMSSender) *Service {
	return &Service{repo: repo, clock: clock, secret: []byte(secret), sms: sms, devEcho: env == "dev"}
}

// RequestOTP generates and stores a code for the phone and texts it via the SMS
// sender. In dev it also returns the code for convenience (the dev SMS sender
// only logs it).
func (s *Service) RequestOTP(ctx context.Context, phone string) (devCode string, err error) {
	code := randomCode()
	if err := s.repo.InsertOTP(ctx, phone, s.hashCode(code), s.clock.Now().Add(otpTTL)); err != nil {
		return "", err
	}
	if err := s.sms.Send(ctx, phone, "Your Yaver verification code is "+code); err != nil {
		return "", err
	}
	if s.devEcho {
		return code, nil
	}
	return "", nil
}

// VerifyOTP checks the code and, on success, issues a session. Returns the raw
// session token (to be set as an httpOnly cookie) and the resolved user.
func (s *Service) VerifyOTP(ctx context.Context, phone, code string) (token string, su domain.SessionUser, err error) {
	otp, found, err := s.repo.LatestLiveOTP(ctx, phone)
	if err != nil {
		return "", domain.SessionUser{}, err
	}
	if !found || otp.Attempts >= maxAttempts {
		return "", domain.SessionUser{}, domain.ErrInvalidOTP
	}
	if !hmac.Equal(otp.CodeHash, s.hashCode(code)) {
		_ = s.repo.IncrementOTPAttempts(ctx, otp.ID)
		return "", domain.SessionUser{}, domain.ErrInvalidOTP
	}
	if err := s.repo.ConsumeOTP(ctx, otp.ID); err != nil {
		return "", domain.SessionUser{}, err
	}

	user, err := s.repo.UpsertUserByPhone(ctx, phone)
	if err != nil {
		return "", domain.SessionUser{}, err
	}
	return s.issueSession(ctx, user)
}

// Register creates an email/password account and logs it in. Returns
// ErrEmailTaken if the email is in use, ErrInvalidCredentials if the password
// is too weak.
func (s *Service) Register(ctx context.Context, email, password, name string) (token string, su domain.SessionUser, err error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(password) < minPasswordLen {
		return "", domain.SessionUser{}, domain.ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", domain.SessionUser{}, err
	}
	user, err := s.repo.CreateUserWithPassword(ctx, email, strings.TrimSpace(name), hash)
	if err != nil {
		return "", domain.SessionUser{}, err // may be ErrEmailTaken
	}
	return s.issueSession(ctx, user)
}

// Login verifies email/password and issues a session. Returns
// ErrInvalidCredentials for any mismatch (never leaks which part was wrong).
func (s *Service) Login(ctx context.Context, email, password string) (token string, su domain.SessionUser, err error) {
	user, hash, found, err := s.repo.UserByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		return "", domain.SessionUser{}, err
	}
	if !found || len(hash) == 0 || bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil {
		return "", domain.SessionUser{}, domain.ErrInvalidCredentials
	}
	return s.issueSession(ctx, user)
}

// issueSession mints a session token for a user and returns it with the resolved
// session identity.
func (s *Service) issueSession(ctx context.Context, user domain.User) (string, domain.SessionUser, error) {
	token := randomToken()
	exp := s.clock.Now().Add(sessionTTL)
	if err := s.repo.CreateSession(ctx, hashToken(token), user.ID, exp); err != nil {
		return "", domain.SessionUser{}, err
	}
	return token, domain.SessionUser{UserID: user.ID, Phone: user.Phone, Email: user.Email, Name: user.Name, ExpiresAt: exp}, nil
}

// Resolve maps a raw session token to its user; found=false when invalid/expired.
func (s *Service) Resolve(ctx context.Context, token string) (domain.SessionUser, bool, error) {
	if token == "" {
		return domain.SessionUser{}, false, nil
	}
	return s.repo.SessionUser(ctx, hashToken(token))
}

// Logout revokes a session.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.DeleteSession(ctx, hashToken(token))
}

func (s *Service) hashCode(code string) []byte {
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(code))
	return m.Sum(nil)
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func randomCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1_000_000))
	return fmt.Sprintf("%06d", n.Int64())
}

func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
