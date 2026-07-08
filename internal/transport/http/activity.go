package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type activityHandler struct {
	log  *slog.Logger
	subs domain.ActivitySubscriber
}

// heartbeat keeps proxies from idling the connection and lets us notice a
// disconnected client between events.
const activityHeartbeat = 25 * time.Second

// stream is the org's live activity feed over Server-Sent Events. It holds the
// request open, writing one SSE frame per event until the client disconnects.
func (h *activityHandler) stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}

	ch, cancel := h.subs.SubscribeActivity(orgFromCtx(r))
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx)
	w.WriteHeader(http.StatusOK)
	// Opening comment flushes headers so the browser's EventSource fires `open`.
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	ticker := time.NewTicker(activityHeartbeat)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case e, ok := <-ch:
			if !ok {
				return
			}
			payload, err := json.Marshal(e)
			if err != nil {
				h.log.Error("marshal activity event", "err", err)
				continue
			}
			// Default `message` event; the discriminator travels in the JSON `type`
			// field so a single client onmessage handler covers every event kind.
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
