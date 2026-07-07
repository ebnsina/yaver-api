package domain

import "errors"

// Typed sentinel errors. Transport maps these to HTTP status codes in one place.
var (
	ErrNotFound            = errors.New("not found")
	ErrInsufficientCredits = errors.New("insufficient credits")
	ErrInvalidPhone        = errors.New("invalid phone number")
	ErrFlowInvalid         = errors.New("invalid flow spec")
)
