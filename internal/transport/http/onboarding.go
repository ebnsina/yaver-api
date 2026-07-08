package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ebnsina/yaver-api/internal/service/onboarding"
)

type onboardingHandler struct {
	log *slog.Logger
	svc *onboarding.Service
}

// ask answers one onboarding question. The dashboard already knows the org's
// setup progress (its Get-started checklist), so it sends it in the body.
func (h *onboardingHandler) ask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Question string `json:"question"`
		Progress struct {
			StoreNamed bool `json:"store_named"`
			HasAPIKey  bool `json:"has_api_key"`
			WebhookSet bool `json:"webhook_set"`
			HasFlow    bool `json:"has_flow"`
		} `json:"progress"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil || body.Question == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "question required"})
		return
	}
	answer, err := h.svc.Ask(r.Context(), onboarding.Progress{
		StoreNamed: body.Progress.StoreNamed,
		HasAPIKey:  body.Progress.HasAPIKey,
		WebhookSet: body.Progress.WebhookSet,
		HasFlow:    body.Progress.HasFlow,
	}, body.Question)
	if err != nil {
		h.log.Error("onboarding ask", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "assistant unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"answer": answer})
}
