package domain

import (
	"context"
	"time"
)

// User is an owned identity (phone-first).
type User struct {
	ID    string
	Phone string
	Email string
	Name  string
}

// OTP is a one-time login code (stored hashed).
type OTP struct {
	ID       string
	CodeHash []byte
	Attempts int
}

// SessionUser is the joined result of resolving a session cookie.
type SessionUser struct {
	UserID    string
	Phone     string
	Email     string
	Name      string
	ExpiresAt time.Time
}

// AuthRepo persists identity, OTPs, and sessions.
type AuthRepo interface {
	UpsertUserByPhone(ctx context.Context, phone string) (User, error)
	InsertOTP(ctx context.Context, phone string, codeHash []byte, expiresAt time.Time) error
	// LatestLiveOTP returns the newest unconsumed, unexpired OTP for a phone.
	// found=false when there is none.
	LatestLiveOTP(ctx context.Context, phone string) (otp OTP, found bool, err error)
	IncrementOTPAttempts(ctx context.Context, otpID string) error
	ConsumeOTP(ctx context.Context, otpID string) error
	// CreateUserWithPassword registers an email/password user. Returns
	// ErrEmailTaken if the email already exists.
	CreateUserWithPassword(ctx context.Context, email, name string, passwordHash []byte) (User, error)
	// UserByEmail loads a user and its bcrypt hash by email (case-insensitive).
	UserByEmail(ctx context.Context, email string) (u User, passwordHash []byte, found bool, err error)
	CreateSession(ctx context.Context, tokenHash []byte, userID string, expiresAt time.Time) error
	// SessionUser resolves a session by token hash. found=false when absent/expired.
	SessionUser(ctx context.Context, tokenHash []byte) (su SessionUser, found bool, err error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
}
