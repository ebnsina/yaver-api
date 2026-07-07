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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ebnsina/yaver-api/internal/adapter/postgres/gen"
	"github.com/ebnsina/yaver-api/internal/domain"
)

type AuthRepo struct {
	q *gen.Queries
}

func NewAuthRepo(pool *pgxpool.Pool) *AuthRepo {
	return &AuthRepo{q: gen.New(pool)}
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
	row, err := r.q.SessionWithUser(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SessionUser{}, false, nil
	}
	if err != nil {
		return domain.SessionUser{}, false, err
	}
	return domain.SessionUser{
		UserID:    row.UserID.String(),
		Phone:     row.Phone,
		Email:     deref(row.Email),
		Name:      deref(row.Name),
		ExpiresAt: row.ExpiresAt,
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
