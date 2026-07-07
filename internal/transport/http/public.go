package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ebnsina/yaver-api/internal/service/apikeys"
	"github.com/ebnsina/yaver-api/internal/service/chat"
)

// publicHandler serves the embeddable widget's cross-origin endpoints. Auth is
// via a publishable (yvr_pk_) key — safe to ship in client-side code.
type publicHandler struct {
	log  *slog.Logger
	keys *apikeys.Service
	chat *chat.Service
}

// cors allows the widget to call these routes from any merchant origin.
// Publishable keys carry no server-side power, so "*" is acceptable here.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type, X-Yaver-Key")
		h.Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *publicHandler) chatSend(w http.ResponseWriter, r *http.Request) {
	org, kind, ok, err := h.keys.Authenticate(r.Context(), r.Header.Get("X-Yaver-Key"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	if !ok || kind != "publishable" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid publishable key"})
		return
	}

	var body struct {
		ConversationID string `json:"conversation_id"`
		Text           string `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil || body.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text required"})
		return
	}
	convID, reply, err := h.chat.Send(r.Context(), org, body.ConversationID, body.Text)
	if err != nil {
		// A bad/foreign conversation_id shouldn't leak; start fresh instead.
		convID, reply, err = h.chat.Send(r.Context(), org, "", body.Text)
		if err != nil {
			h.log.Error("public chat", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"conversation_id": convID, "reply": reply})
}

// chatConfig returns the widget's public appearance (title, welcome, accent),
// resolved from the publishable key. No secrets.
func (h *publicHandler) chatConfig(w http.ResponseWriter, r *http.Request) {
	org, kind, ok, err := h.keys.Authenticate(r.Context(), r.Header.Get("X-Yaver-Key"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	if !ok || kind != "publishable" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid publishable key"})
		return
	}
	cs, err := h.chat.Settings(r.Context(), org)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"title": cs.WidgetTitle, "welcome": cs.Welcome, "accent": cs.Accent,
	})
}

// widget serves the self-contained embed script.
func (h *publicHandler) widget(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(widgetJS))
}
