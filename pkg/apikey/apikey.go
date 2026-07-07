// Package apikey mints and verifies merchant API keys.
//
// A key looks like  yvr_sk_<random>. Only its lookup prefix (indexed) and the
// SHA-256 of the full key are stored, so a DB leak can't reconstruct a usable
// key. Verification is: derive the prefix from the presented key, load the row,
// constant-time compare the hash.
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strings"
)

const (
	scheme    = "yvr_sk_"
	prefixLen = len(scheme) + 8 // stored/indexed prefix
	randLen   = 24              // bytes of entropy
)

// Generate returns (fullKey, lookupPrefix, secretHash). Show fullKey to the user
// exactly once; persist prefix + hash.
func Generate() (full, prefix string, hash []byte) {
	b := make([]byte, randLen)
	_, _ = rand.Read(b)
	full = scheme + base64.RawURLEncoding.EncodeToString(b)
	prefix = full[:prefixLen]
	sum := sha256.Sum256([]byte(full))
	return full, prefix, sum[:]
}

// Prefix derives the lookup prefix from a presented key. ok=false if malformed.
func Prefix(full string) (string, bool) {
	if !strings.HasPrefix(full, scheme) || len(full) < prefixLen {
		return "", false
	}
	return full[:prefixLen], true
}

// Verify constant-time compares a presented key against a stored hash.
func Verify(full string, storedHash []byte) bool {
	sum := sha256.Sum256([]byte(full))
	return subtle.ConstantTimeCompare(sum[:], storedHash) == 1
}
