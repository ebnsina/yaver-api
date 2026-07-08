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
	Env            string // "dev" | "staging" | "prod"
	Port           string
	DatabaseURL    string
	AuthSecret     string // HMAC key for OTP hashing (min 32 bytes recommended)
	Orchestrator   string // "local" | "hatchet"
	EncryptionKey  string // base64 32-byte AES-GCM master key (secrets at rest)
	VoiceProvider  string // "mock" (dev, no telco) | "livekit" — telephony
	LiveKitURL     string // required when VoiceProvider == "livekit"
	LiveKitKey     string
	LiveKitSecret  string
	LiveKitTrunkID string // outbound SIP trunk id
	ChatProvider   string // "builtin" | "openai" | "anthropic" — provider-agnostic AI seam
	AnthropicKey   string // required only when ChatProvider == "anthropic"
	AnthropicModel string // optional; defaults to claude-opus-4-8 in the adapter
	SMSSender      string // "log" (dev) | "twilio" — OTP delivery
	TwilioSID      string // required when SMSSender == "twilio"
	TwilioToken    string
	TwilioFrom     string
	MsgSender      string // "log" | "meta" — messaging (WhatsApp/Messenger) delivery
	EmailSender    string // "log" | "resend" — transactional email delivery
	EmailFrom      string // From address for transactional email
	ResendAPIKey   string // required only when EmailSender == "resend"
	PaymentGateway string // "mock" (dev) | "sslcommerz" — credit top-up gateway
	AppURL         string // API base URL for gateway redirect/IPN callbacks
	WebURL         string // dashboard base URL to send the customer back to after paying
	SSLCzStoreID   string // required when PaymentGateway == "sslcommerz"
	SSLCzStorePass string // required when PaymentGateway == "sslcommerz"
	SSLCzSandbox   bool   // use the SSLCommerz sandbox host
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
		Env:            req("YAVER_ENV"),
		Port:           req("YAVER_PORT"),
		DatabaseURL:    req("YAVER_DATABASE_URL"),
		AuthSecret:     req("YAVER_AUTH_SECRET"),
		Orchestrator:   req("YAVER_ORCHESTRATOR"),
		EncryptionKey:  req("YAVER_ENCRYPTION_KEY"),
		VoiceProvider:  req("YAVER_VOICE_PROVIDER"),
		LiveKitURL:     os.Getenv("YAVER_LIVEKIT_URL"),
		LiveKitKey:     os.Getenv("YAVER_LIVEKIT_API_KEY"),
		LiveKitSecret:  os.Getenv("YAVER_LIVEKIT_API_SECRET"),
		LiveKitTrunkID: os.Getenv("YAVER_LIVEKIT_SIP_TRUNK_ID"),
		SMSSender:      req("YAVER_SMS_SENDER"),
		TwilioSID:      os.Getenv("YAVER_TWILIO_ACCOUNT_SID"),
		TwilioToken:    os.Getenv("YAVER_TWILIO_AUTH_TOKEN"),
		TwilioFrom:     os.Getenv("YAVER_TWILIO_FROM"),
		ChatProvider:   req("YAVER_CHAT_PROVIDER"),
		AnthropicKey:   os.Getenv("YAVER_ANTHROPIC_API_KEY"), // optional; required by the anthropic adapter
		AnthropicModel: os.Getenv("YAVER_ANTHROPIC_MODEL"),   // optional; adapter defaults to claude-opus-4-8
		MsgSender:      req("YAVER_MSG_SENDER"),
		EmailSender:    req("YAVER_EMAIL_SENDER"),
		EmailFrom:      req("YAVER_EMAIL_FROM"),
		ResendAPIKey:   os.Getenv("YAVER_RESEND_API_KEY"), // optional; required by the resend adapter
		PaymentGateway: req("YAVER_PAYMENT_GATEWAY"),
		AppURL:         req("YAVER_APP_URL"),
		WebURL:         req("YAVER_WEB_URL"),
		SSLCzStoreID:   os.Getenv("YAVER_SSLCOMMERZ_STORE_ID"),
		SSLCzStorePass: os.Getenv("YAVER_SSLCOMMERZ_STORE_PASSWD"),
		SSLCzSandbox:   os.Getenv("YAVER_SSLCOMMERZ_SANDBOX") == "true",
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	if cfg.EmailSender == "resend" && cfg.ResendAPIKey == "" {
		return Config{}, fmt.Errorf("YAVER_RESEND_API_KEY is required when YAVER_EMAIL_SENDER=resend")
	}
	if cfg.ChatProvider == "anthropic" && cfg.AnthropicKey == "" {
		return Config{}, fmt.Errorf("YAVER_ANTHROPIC_API_KEY is required when YAVER_CHAT_PROVIDER=anthropic")
	}
	if cfg.SMSSender == "twilio" && (cfg.TwilioSID == "" || cfg.TwilioToken == "" || cfg.TwilioFrom == "") {
		return Config{}, fmt.Errorf("YAVER_TWILIO_ACCOUNT_SID/AUTH_TOKEN/FROM are required when YAVER_SMS_SENDER=twilio")
	}
	if cfg.VoiceProvider == "livekit" && (cfg.LiveKitURL == "" || cfg.LiveKitKey == "" || cfg.LiveKitSecret == "" || cfg.LiveKitTrunkID == "") {
		return Config{}, fmt.Errorf("YAVER_LIVEKIT_URL/API_KEY/API_SECRET/SIP_TRUNK_ID are required when YAVER_VOICE_PROVIDER=livekit")
	}
	if cfg.PaymentGateway == "sslcommerz" && (cfg.SSLCzStoreID == "" || cfg.SSLCzStorePass == "") {
		return Config{}, fmt.Errorf("YAVER_SSLCOMMERZ_STORE_ID and YAVER_SSLCOMMERZ_STORE_PASSWD are required when YAVER_PAYMENT_GATEWAY=sslcommerz")
	}
	return cfg, nil
}
