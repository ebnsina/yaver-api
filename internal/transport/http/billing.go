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
	dev bool // gates the mock top-up / dev-complete endpoints to non-prod
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

// topUp is the dev-only instant grant (no payment). In prod, clients must use
// checkout → gateway → IPN instead.
func (h *billingHandler) topUp(w http.ResponseWriter, r *http.Request) {
	if !h.dev {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "use /v1/billing/checkout to pay"})
		return
	}
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

// checkout starts a real payment and returns the gateway redirect URL. Credits
// land later, on the gateway's IPN confirmation.
func (h *billingHandler) checkout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Credits int `json:"credits"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	url, err := h.svc.Checkout(r.Context(), orgFromCtx(r), body.Credits)
	if err != nil {
		h.log.Error("checkout", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "could not start payment"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirect_url": url})
}

// ipn is the public gateway callback (Instant Payment Notification). It always
// acks with 200 so the gateway doesn't retry a callback we've already processed.
func (h *billingHandler) ipn(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}
	form := make(map[string]string, len(r.PostForm))
	for k := range r.PostForm {
		form[k] = r.PostForm.Get(k)
	}
	if err := h.svc.SettlePayment(r.Context(), form); err != nil {
		h.log.Error("settle payment", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// devPay simulates the gateway + IPN in one click for the mock provider: it
// settles the referenced payment as paid. Dev-only.
func (h *billingHandler) devPay(w http.ResponseWriter, r *http.Request) {
	if !h.dev {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "dev only"})
		return
	}
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing ref"})
		return
	}
	if err := h.svc.SettlePayment(r.Context(), map[string]string{"tran_id": ref, "status": "paid"}); err != nil {
		h.log.Error("dev pay", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "paid"})
}
