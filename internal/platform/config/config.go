// Package config loads typed configuration from the environment (once).
//
// Fail-first: every value is required and read from the environment. Missing
// vars make Load return an error listing all of them, so the process refuses to
// boot rather than running on silent hardcoded defaults. Documented in
// .env.example; supply via real env or a local (gitignored) .env.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Env           string // "dev" | "staging" | "prod"
	Port          string
	DatabaseURL   string
	AuthSecret    string // HMAC key for OTP hashing (min 32 bytes recommended)
	Orchestrator  string // "local" | "hatchet"
	EncryptionKey string // base64 32-byte AES-GCM master key (secrets at rest)
	ChatProvider  string // "builtin" | "openai" | "anthropic" — provider-agnostic AI seam
}

func Load() (Config, error) {
	var missing []string
	req := func(k string) string {
		v := os.Getenv(k)
		if v == "" {
			missing = append(missing, k)
		}
		return v
	}

	cfg := Config{
		Env:           req("YAVER_ENV"),
		Port:          req("YAVER_PORT"),
		DatabaseURL:   req("YAVER_DATABASE_URL"),
		AuthSecret:    req("YAVER_AUTH_SECRET"),
		Orchestrator:  req("YAVER_ORCHESTRATOR"),
		EncryptionKey: req("YAVER_ENCRYPTION_KEY"),
		ChatProvider:  req("YAVER_CHAT_PROVIDER"),
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}
