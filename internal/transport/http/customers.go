package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/service/customers"
)

type customersHandler struct {
	log *slog.Logger
	svc *customers.Service
}

type customerDTO struct {
	ID        string `json:"id"`
	Phone     string `json:"phone"`
	Name      string `json:"name"`
	DND       bool   `json:"dnd"`
	CreatedAt string `json:"created_at"`
}

func (h *customersHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), orgFromCtx(r), 100)
	if err != nil {
		h.log.Error("list customers", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	out := make([]customerDTO, 0, len(list))
	for _, c := range list {
		out = append(out, customerDTO{
			ID: c.ID, Phone: c.Phone, Name: c.Name, DND: c.DND, CreatedAt: c.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"customers": out})
}

func (h *customersHandler) setDND(dnd bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.svc.SetDND(r.Context(), orgFromCtx(r), r.PathValue("id"), dnd); err != nil {
			h.log.Error("set dnd", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"dnd": dnd})
	}
}
