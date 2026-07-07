// Package http is the thin HTTP transport: routing, request/response DTOs, and
// mapping domain errors to status codes. Business logic lives in services.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ebnsina/yaver-api/internal/service/calls"
	"github.com/ebnsina/yaver-api/pkg/phone"
)

type Server struct {
	log   *slog.Logger
	calls *callsHandler
}

// New wires the router. (Phase 0 uses net/http ServeMux; chi + middleware
// arrive with auth/rate-limit in Phase 1.)
func New(log *slog.Logger, callsSvc *calls.Service) http.Handler {
	ch := &callsHandler{log: log, svc: callsSvc}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /v1/dev/test-call", ch.testCall)
	return logRequests(log, mux)
}

type callsHandler struct {
	log *slog.Logger
	svc *calls.Service
}

type testCallRequest struct {
	Phone string `json:"phone"`
	Digit string `json:"digit"` // simulated keypress: "1" | "2" | "3"
}

// testCall runs the order-confirmation IVR against the mock provider with a
// simulated keypress — no telephony. Proves the flow → outcome path end to end.
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
	writeJSON(w, http.StatusOK, map[string]any{
		"call_id": call.ID,
		"status":  out.Status,
		"result":  out.Result,
	})
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
