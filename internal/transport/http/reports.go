package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ebnsina/yaver-api/internal/service/reports"
)

type reportsHandler struct {
	log *slog.Logger
	svc *reports.Service
}

// ask answers a natural-language question over the org's metrics and echoes the
// data the answer was grounded in.
func (h *reportsHandler) ask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Question) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "question required"})
		return
	}
	answer, data, err := h.svc.Ask(r.Context(), orgFromCtx(r), body.Question)
	if err != nil {
		h.log.Error("reports ask", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"answer": answer,
		"data":   overviewDTO(data),
	})
}
