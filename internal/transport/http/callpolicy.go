package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type callPolicyDTO struct {
	WindowStart int    `json:"window_start"`
	WindowEnd   int    `json:"window_end"`
	Timezone    string `json:"timezone"`
	MaxRetries  int    `json:"max_retries"`
}

// getPolicy returns the org's calling policy (defaults if never set).
func (h *callsHandler) getPolicy(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Policy(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("get call policy", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, callPolicyDTO{
		WindowStart: p.WindowStart, WindowEnd: p.WindowEnd, Timezone: p.Timezone, MaxRetries: p.MaxRetries,
	})
}

// savePolicy validates and persists the org's calling policy.
func (h *callsHandler) savePolicy(w http.ResponseWriter, r *http.Request) {
	var body callPolicyDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	p := domain.CallPolicy{
		WindowStart: body.WindowStart, WindowEnd: body.WindowEnd, Timezone: body.Timezone, MaxRetries: body.MaxRetries,
	}
	err := h.svc.SavePolicy(r.Context(), orgFromCtx(r), p)
	switch {
	case errors.Is(err, domain.ErrInvalidCallPolicy):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid policy: window must be start<end within 0-24, valid IANA timezone, retries 0-10"})
	case err != nil:
		h.log.Error("save call policy", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
	}
}
