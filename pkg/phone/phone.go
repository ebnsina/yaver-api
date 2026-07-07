// Package phone normalizes and validates phone numbers to E.164.
// Defaults to Bangladesh (+880) for local-format inputs.
package phone

import (
	"errors"
	"strings"
)

var ErrInvalid = errors.New("invalid phone number")

// NormalizeBD converts common Bangladeshi input formats to E.164 (+8801XXXXXXXXX).
// Accepts: "01712345678", "8801712345678", "+8801712345678".
func NormalizeBD(in string) (string, error) {
	s := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		if r == '+' {
			return r
		}
		return -1
	}, in)

	s = strings.TrimPrefix(s, "+")
	switch {
	case strings.HasPrefix(s, "880") && len(s) == 13:
		// 8801XXXXXXXXX
	case strings.HasPrefix(s, "01") && len(s) == 11:
		s = "880" + s[1:]
	default:
		return "", ErrInvalid
	}
	if s[3] != '1' { // BD mobile numbers are 01X -> 8801X
		return "", ErrInvalid
	}
	return "+" + s, nil
}
