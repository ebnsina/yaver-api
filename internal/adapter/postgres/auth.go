// Package postgres implements domain repositories on top of the sqlc-generated
// queries (internal/adapter/postgres/gen). It converts between DB types
// (uuid.UUID, nullable pointers) and the plain domain types.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

// AuthRepo mixes sqlc-generated queries (phone/OTP paths) with a few raw pgx
// queries for the email/password path — where users may have a NULL phone that
// the phone-typed generated columns can't scan.
type AuthRepo struct {
	q    *gen.Queries
	pool *pgxpool.Pool
}

func NewAuthRepo(pool *pgxpool.Pool) *AuthRepo {
	return &AuthRepo{q: gen.New(pool), pool: pool}
}

// CreateUserWithPassword inserts an email/password user.
func (r *AuthRepo) CreateUserWithPassword(ctx context.Context, email, name string, passwordHash []byte) (domain.User, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING id`,
		email, name, passwordHash).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return domain.User{}, domain.ErrEmailTaken
	}
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: id.String(), Email: email, Name: name}, nil
}

// UserByEmail loads a user and bcrypt hash by email (case-insensitive).
func (r *AuthRepo) UserByEmail(ctx context.Context, email string) (domain.User, []byte, bool, error) {
	var (
		id              uuid.UUID
		phone, em, name *string
		hash            []byte
	)
	err := r.pool.QueryRow(ctx,
		`SELECT id, phone, email, name, password_hash FROM users WHERE lower(email) = lower($1)`, email).
		Scan(&id, &phone, &em, &name, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, nil, false, nil
	}
	if err != nil {
		return domain.User{}, nil, false, err
	}
	return domain.User{ID: id.String(), Phone: deref(phone), Email: deref(em), Name: deref(name)}, hash, true, nil
}

func (r *AuthRepo) UpsertUserByPhone(ctx context.Context, phone string) (domain.User, error) {
	u, err := r.q.UpsertUserByPhone(ctx, phone)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: u.ID.String(), Phone: u.Phone, Email: deref(u.Email), Name: deref(u.Name)}, nil
}

func (r *AuthRepo) InsertOTP(ctx context.Context, phone string, codeHash []byte, expiresAt time.Time) error {
	return r.q.InsertOTP(ctx, gen.InsertOTPParams{Phone: phone, CodeHash: codeHash, ExpiresAt: expiresAt})
}

func (r *AuthRepo) LatestLiveOTP(ctx context.Context, phone string) (domain.OTP, bool, error) {
	o, err := r.q.LatestLiveOTP(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OTP{}, false, nil
	}
	if err != nil {
		return domain.OTP{}, false, err
	}
	return domain.OTP{ID: o.ID.String(), CodeHash: o.CodeHash, Attempts: int(o.Attempts)}, true, nil
}

func (r *AuthRepo) IncrementOTPAttempts(ctx context.Context, otpID string) error {
	id, err := uuid.Parse(otpID)
	if err != nil {
		return err
	}
	return r.q.IncrementOTPAttempts(ctx, id)
}

func (r *AuthRepo) ConsumeOTP(ctx context.Context, otpID string) error {
	id, err := uuid.Parse(otpID)
	if err != nil {
		return err
	}
	return r.q.ConsumeOTP(ctx, id)
}

func (r *AuthRepo) CreateSession(ctx context.Context, tokenHash []byte, userID string, expiresAt time.Time) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	_, err = r.q.CreateSession(ctx, gen.CreateSessionParams{TokenHash: tokenHash, UserID: uid, ExpiresAt: expiresAt})
	return err
}

func (r *AuthRepo) SessionUser(ctx context.Context, tokenHash []byte) (domain.SessionUser, bool, error) {
	// Raw (not sqlc) so a NULL phone — email/password users — scans cleanly.
	var (
		userID             uuid.UUID
		phone, email, name *string
		exp                time.Time
	)
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.phone, u.email, u.name, s.expires_at
		 FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = $1 AND s.expires_at > now()`, tokenHash).
		Scan(&userID, &phone, &email, &name, &exp)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SessionUser{}, false, nil
	}
	if err != nil {
		return domain.SessionUser{}, false, err
	}
	return domain.SessionUser{
		UserID:    userID.String(),
		Phone:     deref(phone),
		Email:     deref(email),
		Name:      deref(name),
		ExpiresAt: exp,
	}, true, nil
}

func (r *AuthRepo) DeleteSession(ctx context.Context, tokenHash []byte) error {
	return r.q.DeleteSession(ctx, tokenHash)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
