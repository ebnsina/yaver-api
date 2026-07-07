package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ebnsina/yaver-api/internal/service/webhooks"
)

type webhookHandler struct {
	log *slog.Logger
	svc *webhooks.Service
}

// getEndpoint returns the org's current webhook config (never the secret).
func (h *webhookHandler) getEndpoint(w http.ResponseWriter, r *http.Request) {
	url, events, found, err := h.svc.Get(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("get webhook", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "events": []string{}})
		return
	}
	if events == nil {
		events = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"configured": true, "url": url, "events": events})
}

// setEndpoint configures the authenticated org's webhook URL and returns the
// signing secret once.
func (h *webhookHandler) setEndpoint(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url required"})
		return
	}
	secret, err := h.svc.SetEndpoint(r.Context(), orgFromCtx(r), body.URL, body.Events)
	if err != nil {
		h.log.Error("set webhook", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"secret": secret})
}
