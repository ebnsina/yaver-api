package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/service/messaging"
)

type channelsHandler struct {
	log *slog.Logger
	svc *messaging.Service
}

type channelDTO struct {
	Type       string `json:"type"`
	ExternalID string `json:"external_id"`
	CreatedAt  string `json:"created_at"`
}

func (h *channelsHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("list channels", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	out := make([]channelDTO, 0, len(list))
	for _, c := range list {
		out = append(out, channelDTO{Type: c.Type, ExternalID: c.ExternalID, CreatedAt: c.CreatedAt.Format(time.RFC3339)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"channels": out})
}

func (h *channelsHandler) connect(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type        string `json:"type"`
		ExternalID  string `json:"external_id"`
		AccessToken string `json:"access_token"`
		VerifyToken string `json:"verify_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		(body.Type != "whatsapp" && body.Type != "messenger") ||
		body.ExternalID == "" || body.AccessToken == "" || body.VerifyToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type, external_id, access_token, verify_token required"})
		return
	}
	if err := h.svc.Connect(r.Context(), orgFromCtx(r), body.Type, body.ExternalID, body.AccessToken, body.VerifyToken); err != nil {
		h.log.Error("connect channel", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "connected"})
}

func (h *channelsHandler) disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Disconnect(r.Context(), orgFromCtx(r), r.PathValue("type")); err != nil {
		h.log.Error("disconnect channel", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

// --- Meta webhook (inbound) ---------------------------------------------------

type metaWebhookHandler struct {
	log *slog.Logger
	svc *messaging.Service
}

// verify handles Meta's GET subscription handshake.
func (h *metaWebhookHandler) verify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("hub.mode") == "subscribe" && h.svc.VerifyChallenge(r.Context(), q.Get("hub.verify_token")) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(q.Get("hub.challenge")))
		return
	}
	w.WriteHeader(http.StatusForbidden)
}

// metaPayload covers the subset of WhatsApp + Messenger webhook shapes we need.
type metaPayload struct {
	Entry []struct {
		ID      string `json:"id"` // Messenger page id
		Changes []struct {
			Value struct {
				Metadata struct {
					PhoneNumberID string `json:"phone_number_id"` // WhatsApp
				} `json:"metadata"`
				Messages []struct {
					From string `json:"from"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
		Messaging []struct {
			Sender  struct{ ID string } `json:"sender"`
			Message struct {
				Text string `json:"text"`
			} `json:"message"`
		} `json:"messaging"`
	} `json:"entry"`
}

// receive parses inbound messages and routes each through the assistant.
func (h *metaWebhookHandler) receive(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var p metaPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		w.WriteHeader(http.StatusOK) // ack anything; never make Meta retry on parse
		return
	}
	for _, e := range p.Entry {
		// WhatsApp
		for _, c := range e.Changes {
			to := c.Value.Metadata.PhoneNumberID
			for _, m := range c.Value.Messages {
				if m.Text.Body != "" {
					h.route(r, to, m.From, m.Text.Body)
				}
			}
		}
		// Messenger
		for _, m := range e.Messaging {
			if m.Message.Text != "" {
				h.route(r, e.ID, m.Sender.ID, m.Message.Text)
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (h *metaWebhookHandler) route(r *http.Request, to, from, text string) {
	if err := h.svc.HandleInbound(r.Context(), to, from, text); err != nil {
		h.log.Error("inbound route", "err", err)
	}
}
