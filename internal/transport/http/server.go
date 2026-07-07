// Package http is the thin HTTP transport: routing, request/response DTOs, and
// mapping domain errors to status codes. Business logic lives in services.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/auth"
	"github.com/ebnsina/yaver-api/internal/service/calls"
	"github.com/ebnsina/yaver-api/pkg/phone"
)

// New wires the router. (Phase 0 uses net/http ServeMux; chi + richer middleware
// arrive with rate-limit / API keys in Phase 1.)
func New(log *slog.Logger, env string, authSvc *auth.Service, callsSvc *calls.Service, orch domain.Orchestrator) http.Handler {
	ah := &authHandler{log: log, svc: authSvc, secure: env != "dev"}
	ch := &callsHandler{log: log, svc: callsSvc, orch: orch}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth (phone-OTP + cookie sessions).
	mux.HandleFunc("POST /v1/auth/otp/request", ah.requestOTP)
	mux.HandleFunc("POST /v1/auth/otp/verify", ah.verifyOTP)
	mux.HandleFunc("POST /v1/auth/logout", ah.logout)
	mux.Handle("GET /v1/me", ah.requireAuth(http.HandlerFunc(ah.me)))

	// Dev endpoints (no telco): simulate a full flow, or enqueue place_call.
	mux.HandleFunc("POST /v1/dev/test-call", ch.testCall)
	mux.HandleFunc("POST /v1/dev/place-call", ch.placeCall)

	return logRequests(log, mux)
}

type callsHandler struct {
	log  *slog.Logger
	svc  *calls.Service
	orch domain.Orchestrator
}

type testCallRequest struct {
	Phone string `json:"phone"`
	Digit string `json:"digit"` // simulated keypress: "1" | "2" | "3"
}

// testCall runs the order-confirmation IVR against the mock provider with a
// simulated keypress — no telephony. Proves the flow → outcome path.
func (h *callsHandler) testCall(w http.ResponseWriter, r *http.Request) {
	var req testCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	e164, err := phone.NormalizeBD(req.Phone)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid phone"})
		return
	}
	if req.Digit == "" {
		req.Digit = "1"
	}
	out, call, err := h.svc.RunTestCall(r.Context(), "org_demo", e164, req.Digit, demoOrderConfirmFlow())
	if err != nil {
		h.log.Error("test-call failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"call_id": call.ID, "status": out.Status, "result": out.Result})
}

// placeCall enqueues a place_call job through the orchestrator (async path).
func (h *callsHandler) placeCall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	e164, err := phone.NormalizeBD(req.Phone)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid phone"})
		return
	}
	if err := h.orch.EnqueuePlaceCall(r.Context(), domain.PlaceCallInput{
		OrgID:   "org_demo",
		ToPhone: e164,
		FlowID:  "flow_demo_order_confirm",
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("http", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
