package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/flows"
)

type flowsHandler struct {
	log *slog.Logger
	svc *flows.Service
}

type flowSummaryDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
	Channel string `json:"channel"`
	Type    string `json:"type"`
	Active  bool   `json:"active"`
}

func (h *flowsHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("list flows", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	out := make([]flowSummaryDTO, 0, len(list))
	for _, f := range list {
		out = append(out, flowSummaryDTO{
			ID: string(f.ID), Name: f.Name, Version: f.Version,
			Channel: string(f.Channel), Type: string(f.Type), Active: f.Active,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"flows": out})
}

// create adds a new flow (no-code builder / templates) and returns its id.
func (h *flowsHandler) create(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var body struct {
		Name    string          `json:"name"`
		Channel string          `json:"channel"`
		Type    string          `json:"type"`
		Locale  string          `json:"locale"`
		Spec    json.RawMessage `json:"spec"`
	}
	if err := json.Unmarshal(raw, &body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	id, err := h.svc.Create(r.Context(), orgFromCtx(r), domain.NewFlow{
		Name:    body.Name,
		Channel: domain.Channel(body.Channel),
		Type:    domain.FlowType(body.Type),
		Locale:  body.Locale,
		Spec:    body.Spec,
	})
	switch {
	case errors.Is(err, domain.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "a flow with that name already exists"})
	case errors.Is(err, domain.ErrFlowInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flow (name, type, or spec)"})
	case err != nil:
		h.log.Error("create flow", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusCreated, map[string]string{"id": string(id)})
	}
}

func (h *flowsHandler) get(w http.ResponseWriter, r *http.Request) {
	fd, err := h.svc.Get(r.Context(), orgFromCtx(r), domain.FlowID(r.PathValue("id")))
	if errors.Is(err, domain.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		h.log.Error("get flow", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      string(fd.ID),
		"name":    fd.Name,
		"version": fd.Version,
		"channel": string(fd.Channel),
		"type":    string(fd.Type),
		"locale":  fd.Locale,
		"spec":    json.RawMessage(fd.Spec),
	})
}

func (h *flowsHandler) update(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var body struct {
		Spec json.RawMessage `json:"spec"`
	}
	if err := json.Unmarshal(raw, &body); err != nil || len(body.Spec) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "spec required"})
		return
	}
	err := h.svc.UpdateSpec(r.Context(), orgFromCtx(r), domain.FlowID(r.PathValue("id")), body.Spec)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, domain.ErrFlowInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flow spec"})
	case err != nil:
		h.log.Error("update flow", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
	}
}
