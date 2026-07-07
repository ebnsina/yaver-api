-- name: UpsertUserByPhone :one
INSERT INTO users (phone) VALUES ($1)
ON CONFLICT (phone) DO UPDATE SET phone = EXCLUDED.phone
RETURNING *;

-- name: InsertOTP :exec
INSERT INTO otp_codes (phone, code_hash, expires_at) VALUES ($1, $2, $3);

-- name: LatestLiveOTP :one
SELECT * FROM otp_codes
WHERE phone = $1 AND consumed_at IS NULL AND expires_at > now()
ORDER BY created_at DESC
LIMIT 1;

-- name: IncrementOTPAttempts :exec
UPDATE otp_codes SET attempts = attempts + 1 WHERE id = $1;

-- name: ConsumeOTP :exec
UPDATE otp_codes SET consumed_at = now() WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)
RETURNING *;

-- name: SessionWithUser :one
SELECT s.user_id, s.expires_at, u.phone, u.email, u.name
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;
