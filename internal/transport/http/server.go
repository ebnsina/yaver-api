// Package http is the thin HTTP transport: routing, request/response DTOs, and
// mapping domain errors to status codes. Business logic lives in services.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/apikeys"
	"github.com/ebnsina/yaver-api/internal/service/auth"
	"github.com/ebnsina/yaver-api/internal/service/calls"
	"github.com/ebnsina/yaver-api/internal/service/campaigns"
	"github.com/ebnsina/yaver-api/internal/service/chat"
	"github.com/ebnsina/yaver-api/internal/service/customers"
	"github.com/ebnsina/yaver-api/internal/service/flows"
	"github.com/ebnsina/yaver-api/internal/service/ingest"
	"github.com/ebnsina/yaver-api/internal/service/webhooks"
	"github.com/ebnsina/yaver-api/internal/transport/openapi"
	"github.com/ebnsina/yaver-api/pkg/phone"
)

// New wires the router. (Phase 0 uses net/http ServeMux; chi + richer middleware
// arrive with rate-limit in Phase 1.)
func New(log *slog.Logger, env string, authSvc *auth.Service, orgStore domain.OrgStore, callsSvc *calls.Service, flowsSvc *flows.Service, custSvc *customers.Service, campSvc *campaigns.Service, chatSvc *chat.Service, keysSvc *apikeys.Service, ingestSvc *ingest.Service, webhooksSvc *webhooks.Service, orch domain.Orchestrator) http.Handler {
	dev := env == "dev"
	ah := &authHandler{log: log, svc: authSvc, orgs: orgStore, secure: !dev}
	ch := &callsHandler{log: log, svc: callsSvc, orch: orch}
	fh := &flowsHandler{log: log, svc: flowsSvc}
	cuh := &customersHandler{log: log, svc: custSvc}
	cah := &campaignsHandler{log: log, svc: campSvc}
	chh := &chatHandler{log: log, svc: chatSvc}
	ph := &publicHandler{log: log, keys: keysSvc, chat: chatSvc}
	ih := &ingestHandler{log: log, keys: keysSvc, ingest: ingestSvc}
	wh := &webhookHandler{log: log, svc: webhooksSvc}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(openapi.Spec)
	})

	// Auth (phone-OTP + cookie sessions).
	mux.HandleFunc("POST /v1/auth/otp/request", ah.requestOTP)
	mux.HandleFunc("POST /v1/auth/otp/verify", ah.verifyOTP)
	mux.HandleFunc("POST /v1/auth/logout", ah.logout)
	mux.Handle("GET /v1/me", ah.requireAuth(http.HandlerFunc(ah.me)))
	mux.Handle("GET /v1/calls", ah.requireAuth(http.HandlerFunc(ch.listCalls)))
	mux.Handle("GET /v1/calls/{id}", ah.requireAuth(http.HandlerFunc(ch.getCall)))
	mux.Handle("GET /v1/analytics/summary", ah.requireAuth(http.HandlerFunc(ch.summary)))

	// Customers + DND.
	mux.Handle("GET /v1/customers", ah.requireAuth(http.HandlerFunc(cuh.list)))
	mux.Handle("POST /v1/customers/{id}/dnd", ah.requireAuth(cuh.setDND(true)))
	mux.Handle("DELETE /v1/customers/{id}/dnd", ah.requireAuth(cuh.setDND(false)))

	// Chat channel.
	mux.Handle("POST /v1/chat/messages", ah.requireAuth(http.HandlerFunc(chh.send)))
	mux.Handle("GET /v1/chat/conversations", ah.requireAuth(http.HandlerFunc(chh.list)))
	mux.Handle("GET /v1/chat/conversations/{id}", ah.requireAuth(http.HandlerFunc(chh.messages)))

	// Campaigns (bulk outbound).
	mux.Handle("GET /v1/campaigns", ah.requireAuth(http.HandlerFunc(cah.list)))
	mux.Handle("POST /v1/campaigns", ah.requireAuth(http.HandlerFunc(cah.create)))
	mux.Handle("POST /v1/campaigns/{id}/start", ah.requireAuth(http.HandlerFunc(cah.start)))

	// Flows (no-code builder).
	mux.Handle("GET /v1/flows", ah.requireAuth(http.HandlerFunc(fh.list)))
	mux.Handle("GET /v1/flows/{id}", ah.requireAuth(http.HandlerFunc(fh.get)))
	mux.Handle("PUT /v1/flows/{id}", ah.requireAuth(http.HandlerFunc(fh.update)))

	// Merchant ingest (X-API-Key resolves the org).
	mux.Handle("POST /v1/events", ih.requireAPIKey(http.HandlerFunc(ih.postEvent)))

	// Authenticated actions on the caller's org (org resolved from the session).
	mux.Handle("PUT /v1/settings/org", ah.requireAuth(http.HandlerFunc(ah.renameOrg)))
	mux.Handle("GET /v1/settings/api-keys", ah.requireAuth(http.HandlerFunc(ih.listKeys)))
	mux.Handle("POST /v1/settings/api-keys", ah.requireAuth(http.HandlerFunc(ih.mintKey)))
	mux.Handle("POST /v1/settings/publishable-key", ah.requireAuth(http.HandlerFunc(ih.mintPublishableKey)))

	// Public widget surface (cross-origin, publishable-key auth).
	mux.Handle("/public/chat/messages", cors(http.HandlerFunc(ph.chatSend)))
	mux.Handle("GET /widget.js", cors(http.HandlerFunc(ph.widget)))
	mux.Handle("GET /v1/settings/webhook", ah.requireAuth(http.HandlerFunc(wh.getEndpoint)))
	mux.Handle("POST /v1/settings/webhook", ah.requireAuth(http.HandlerFunc(wh.setEndpoint)))
	mux.Handle("POST /v1/dev/test-call", ah.requireAuth(http.HandlerFunc(ch.testCall)))
	mux.Handle("POST /v1/dev/place-call", ah.requireAuth(http.HandlerFunc(ch.placeCall)))

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
	out, call, err := h.svc.RunTestCall(r.Context(), orgFromCtx(r), e164, req.Digit, "order_confirm")
	if errors.Is(err, domain.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no active order_confirm flow"})
		return
	}
	if err != nil {
		h.log.Error("test-call failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"call_id": call.ID, "status": out.Status, "result": out.Result})
}

type callDTO struct {
	ID        string `json:"id"`
	Direction string `json:"direction"`
	Status    string `json:"status"`
	Result    string `json:"result"`
	CreatedAt string `json:"created_at"`
}

// listCalls returns the org's recent calls. Phase 0 scopes to org_demo until
// user→org onboarding exists.
func (h *callsHandler) listCalls(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), orgFromCtx(r), 50)
	if err != nil {
		h.log.Error("list calls", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	out := make([]callDTO, 0, len(list))
	for _, c := range list {
		out = append(out, callDTO{
			ID:        string(c.ID),
			Direction: string(c.Direction),
			Status:    string(c.Status),
			Result:    c.Result,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"calls": out})
}

type callDetailDTO struct {
	ID             string `json:"id"`
	Direction      string `json:"direction"`
	Status         string `json:"status"`
	Result         string `json:"result"`
	FlowID         string `json:"flow_id"`
	ProviderCallID string `json:"provider_call_id"`
	CreatedAt      string `json:"created_at"`
}

// getCall returns one call, org-scoped (404 if not the caller's).
func (h *callsHandler) getCall(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Get(r.Context(), orgFromCtx(r), domain.CallID(r.PathValue("id")))
	if errors.Is(err, domain.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		h.log.Error("get call", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, callDetailDTO{
		ID:             string(c.ID),
		Direction:      string(c.Direction),
		Status:         string(c.Status),
		Result:         c.Result,
		FlowID:         string(c.FlowID),
		ProviderCallID: string(c.ProviderCallID),
		CreatedAt:      c.CreatedAt.Format(time.RFC3339),
	})
}

// summary returns dashboard metrics for the org.
func (h *callsHandler) summary(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Summary(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("summary", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	rate := 0
	if s.Total > 0 {
		rate = int(float64(s.Confirmed) / float64(s.Total) * 100)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total":             s.Total,
		"confirmed":         s.Confirmed,
		"cancelled":         s.Cancelled,
		"today":             s.Today,
		"confirmation_rate": rate,
	})
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
		OrgID:   orgFromCtx(r),
		ToPhone: e164,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

// orgFromCtx reads the org injected by requireAuth (session) or requireAPIKey.
func orgFromCtx(r *http.Request) domain.OrgID {
	org, _ := r.Context().Value(orgKey).(domain.OrgID)
	return org
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
