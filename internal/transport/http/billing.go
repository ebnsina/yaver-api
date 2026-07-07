package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/service/billing"
)

type billingHandler struct {
	log *slog.Logger
	svc *billing.Service
}

type ledgerEntryDTO struct {
	Delta     int    `json:"delta"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

func (h *billingHandler) get(w http.ResponseWriter, r *http.Request) {
	org := orgFromCtx(r)
	bal, err := h.svc.Balance(r.Context(), org)
	if err != nil {
		h.log.Error("balance", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	entries, err := h.svc.Ledger(r.Context(), org, 20)
	if err != nil {
		h.log.Error("ledger", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	out := make([]ledgerEntryDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, ledgerEntryDTO{Delta: e.Delta, Reason: e.Reason, CreatedAt: e.CreatedAt.Format(time.RFC3339)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"balance": bal, "ledger": out})
}

func (h *billingHandler) topUp(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Amount int `json:"amount"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Amount <= 0 {
		body.Amount = 100
	}
	bal, err := h.svc.TopUp(r.Context(), orgFromCtx(r), body.Amount)
	if err != nil {
		h.log.Error("topup", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"balance": bal})
}
