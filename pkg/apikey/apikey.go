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
	SecretScheme      = "yvr_sk_" // server-side keys
	PublishableScheme = "yvr_pk_" // embeddable (browser widget) keys
	schemeLen         = len(SecretScheme)
	prefixLen         = schemeLen + 8 // stored/indexed prefix (both schemes are 7 chars)
	randLen           = 24            // bytes of entropy
)

// Generate returns a secret key (fullKey, lookupPrefix, secretHash). Show fullKey
// to the user exactly once; persist prefix + hash.
func Generate() (full, prefix string, hash []byte) { return gen(SecretScheme) }

// GeneratePublishable returns a publishable (yvr_pk_) key, safe to embed client-side.
func GeneratePublishable() (full, prefix string, hash []byte) { return gen(PublishableScheme) }

func gen(scheme string) (full, prefix string, hash []byte) {
	b := make([]byte, randLen)
	_, _ = rand.Read(b)
	full = scheme + base64.RawURLEncoding.EncodeToString(b)
	prefix = full[:prefixLen]
	sum := sha256.Sum256([]byte(full))
	return full, prefix, sum[:]
}

// Prefix derives the lookup prefix from a presented key (either scheme).
// ok=false if malformed.
func Prefix(full string) (string, bool) {
	if !strings.HasPrefix(full, "yvr_") || len(full) < prefixLen {
		return "", false
	}
	return full[:prefixLen], true
}

// Verify constant-time compares a presented key against a stored hash.
func Verify(full string, storedHash []byte) bool {
	sum := sha256.Sum256([]byte(full))
	return subtle.ConstantTimeCompare(sum[:], storedHash) == 1
}
