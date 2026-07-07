// Package id generates URL-safe random identifiers with a short type prefix.
package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a prefixed random id, e.g. New("call") -> "call_1a2b3c...".
func New(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
