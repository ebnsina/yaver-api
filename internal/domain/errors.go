package domain

import "errors"

// Typed sentinel errors. Transport maps these to HTTP status codes in one place.
var (
	ErrNotFound            = errors.New("not found")
	ErrInsufficientCredits = errors.New("insufficient credits")
	ErrInvalidPhone        = errors.New("invalid phone number")
	ErrFlowInvalid         = errors.New("invalid flow spec")
	ErrInvalidOTP          = errors.New("invalid or expired code")
	ErrEmailTaken          = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrOutsideCallWindow   = errors.New("outside the allowed calling window")
	ErrInvalidCallPolicy   = errors.New("invalid call policy")
	ErrConflict            = errors.New("already exists")
)
