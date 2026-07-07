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
