package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/campaigns"
)

type campaignsHandler struct {
	log *slog.Logger
	svc *campaigns.Service
}

type campaignDTO struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	TargetCount int     `json:"target_count"`
	CreatedAt   string  `json:"created_at"`
	StartedAt   *string `json:"started_at"`
}

func toCampaignDTO(c domain.Campaign) campaignDTO {
	var started *string
	if c.StartedAt != nil {
		s := c.StartedAt.Format(time.RFC3339)
		started = &s
	}
	return campaignDTO{
		ID: c.ID, Name: c.Name, Status: c.Status, TargetCount: c.TargetCount,
		CreatedAt: c.CreatedAt.Format(time.RFC3339), StartedAt: started,
	}
}

func (h *campaignsHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	id, err := h.svc.Create(r.Context(), orgFromCtx(r), body.Name)
	if err != nil {
		h.log.Error("create campaign", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *campaignsHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("list campaigns", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	out := make([]campaignDTO, 0, len(list))
	for _, c := range list {
		out = append(out, toCampaignDTO(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaigns": out})
}

func (h *campaignsHandler) start(w http.ResponseWriter, r *http.Request) {
	queued, err := h.svc.Start(r.Context(), orgFromCtx(r), r.PathValue("id"))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case err != nil:
		h.log.Error("start campaign", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusOK, map[string]int{"queued": queued})
	}
}
