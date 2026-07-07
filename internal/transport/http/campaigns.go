package http

import (
	"encoding/json"
	"errors"
	"io"
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
	ScheduledAt *string `json:"scheduled_at"`
}

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func toCampaignDTO(c domain.Campaign) campaignDTO {
	return campaignDTO{
		ID: c.ID, Name: c.Name, Status: c.Status, TargetCount: c.TargetCount,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		StartedAt: rfc3339Ptr(c.StartedAt), ScheduledAt: rfc3339Ptr(c.ScheduledAt),
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

// importRecipients accepts a CSV body (phone[,name] per row) and adds the valid,
// deduped recipients to the campaign.
func (h *campaignsHandler) importRecipients(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 5<<20)) // 5MB cap
	added, err := h.svc.ImportRecipients(r.Context(), orgFromCtx(r), r.PathValue("id"), string(raw))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, domain.ErrFlowInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed CSV"})
	case err != nil:
		h.log.Error("import recipients", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusOK, map[string]int{"added": added})
	}
}

// schedule sets a draft campaign to auto-start at a future time (RFC3339).
func (h *campaignsHandler) schedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		At string `json:"at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	at, perr := time.Parse(time.RFC3339, body.At)
	if perr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at must be RFC3339"})
		return
	}
	err := h.svc.Schedule(r.Context(), orgFromCtx(r), r.PathValue("id"), at)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, domain.ErrFlowInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "schedule time must be in the future"})
	case err != nil:
		h.log.Error("schedule campaign", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "scheduled", "at": at.Format(time.RFC3339)})
	}
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
