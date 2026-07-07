package http

import (
	"log/slog"
	"net/http"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/analytics"
)

type analyticsHandler struct {
	log *slog.Logger
	svc *analytics.Service
}

// pct returns part/total as a whole-number percentage (0 when total is 0).
func pct(part, total int) int {
	if total <= 0 {
		return 0
	}
	return int(float64(part) / float64(total) * 100)
}

// overviewDTO serializes the cross-channel rollup with headline rates
// pre-computed for the UI. Shared by the analytics and reports handlers.
func overviewDTO(o domain.AnalyticsOverview) map[string]any {
	return map[string]any{
		"calls": map[string]any{
			"total":             o.Calls.Total,
			"confirmed":         o.Calls.Confirmed,
			"cancelled":         o.Calls.Cancelled,
			"today":             o.Calls.Today,
			"confirmation_rate": pct(o.Calls.Confirmed, o.Calls.Total),
		},
		"conversations": map[string]any{
			"total":           o.Conversations.Total,
			"today":           o.Conversations.Today,
			"resolved":        o.Conversations.Resolved,
			"pending":         o.Conversations.Pending,
			"sale":            o.Conversations.Sale,
			"needs_human":     o.Conversations.NeedsHuman,
			"resolution_rate": pct(o.Conversations.Resolved, o.Conversations.Total),
		},
		"credits": map[string]any{
			"balance":     o.Credits.Balance,
			"spent_today": o.Credits.SpentToday,
			"spent_30d":   o.Credits.Spent30d,
		},
	}
}

// overview returns the cross-channel dashboard rollup.
func (h *analyticsHandler) overview(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.Overview(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("analytics overview", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, overviewDTO(o))
}
