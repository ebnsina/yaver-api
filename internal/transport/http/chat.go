package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/chat"
	"github.com/ebnsina/yaver-api/internal/service/messaging"
)

type chatHandler struct {
	log *slog.Logger
	svc *chat.Service
	msg *messaging.Service
}

type conversationDTO struct {
	ID           string `json:"id"`
	Channel      string `json:"channel"`
	Customer     string `json:"customer"`
	Status       string `json:"status"`
	LastMessage  string `json:"last_message"`
	MessageCount int    `json:"message_count"`
	UpdatedAt    string `json:"updated_at"`
}

type messageDTO struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type insightDTO struct {
	Summary    string `json:"summary"`
	Outcome    string `json:"outcome"`
	Sentiment  string `json:"sentiment"`
	NextAction string `json:"next_action"`
	CreatedAt  string `json:"created_at"`
}

func insightToDTO(in domain.ConversationInsight) insightDTO {
	return insightDTO{
		Summary:    in.Summary,
		Outcome:    in.Outcome,
		Sentiment:  in.Sentiment,
		NextAction: in.NextAction,
		CreatedAt:  in.CreatedAt.Format(time.RFC3339),
	}
}

// send posts a user message and returns the assistant reply.
func (h *chatHandler) send(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ConversationID string `json:"conversation_id"`
		Text           string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text required"})
		return
	}
	convID, reply, err := h.svc.Send(r.Context(), orgFromCtx(r), body.ConversationID, body.Text)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case err != nil:
		h.log.Error("chat send", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"conversation_id": convID, "reply": reply})
	}
}

func (h *chatHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("chat list", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	out := make([]conversationDTO, 0, len(list))
	for _, c := range list {
		customer := c.ExternalUser
		if customer == "" {
			customer = "Website visitor"
		}
		out = append(out, conversationDTO{
			ID: c.ID, Channel: c.Channel, Customer: customer, Status: c.Status,
			LastMessage: c.LastMessage, MessageCount: c.MessageCount, UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": out})
}

type chatSettingsDTO struct {
	Instructions string `json:"instructions"`
	WidgetTitle  string `json:"widget_title"`
	Welcome      string `json:"welcome"`
	Accent       string `json:"accent"`
}

func (h *chatHandler) getSettings(w http.ResponseWriter, r *http.Request) {
	cs, err := h.svc.Settings(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("chat settings", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, chatSettingsDTO{
		Instructions: cs.Instructions, WidgetTitle: cs.WidgetTitle, Welcome: cs.Welcome, Accent: cs.Accent,
	})
}

func (h *chatHandler) saveSettings(w http.ResponseWriter, r *http.Request) {
	var body chatSettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	// Fall back to defaults for blank display fields.
	def := domain.DefaultChatSettings()
	cs := domain.ChatSettings{Instructions: body.Instructions, WidgetTitle: body.WidgetTitle, Welcome: body.Welcome, Accent: body.Accent}
	if cs.WidgetTitle == "" {
		cs.WidgetTitle = def.WidgetTitle
	}
	if cs.Welcome == "" {
		cs.Welcome = def.Welcome
	}
	if cs.Accent == "" {
		cs.Accent = def.Accent
	}
	if err := h.svc.SaveSettings(r.Context(), orgFromCtx(r), cs); err != nil {
		h.log.Error("save chat settings", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (h *chatHandler) messages(w http.ResponseWriter, r *http.Request) {
	msgs, err := h.svc.Messages(r.Context(), orgFromCtx(r), r.PathValue("id"))
	if errors.Is(err, domain.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		h.log.Error("chat messages", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	out := make([]messageDTO, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, messageDTO{Role: m.Role, Content: m.Content, CreatedAt: m.CreatedAt.Format(time.RFC3339)})
	}
	body := map[string]any{"messages": out}
	// Attach the cached AI insight if one has been generated.
	if in, found, err := h.svc.Insight(r.Context(), orgFromCtx(r), r.PathValue("id")); err != nil {
		h.log.Error("chat insight", "err", err)
	} else if found {
		dto := insightToDTO(in)
		body["insight"] = &dto
	}
	writeJSON(w, http.StatusOK, body)
}

// setStatus transitions a conversation (open | handling | closed) — e.g. a human
// taking it over or closing it out.
func (h *chatHandler) setStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	err := h.svc.SetStatus(r.Context(), orgFromCtx(r), r.PathValue("id"), body.Status)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, domain.ErrFlowInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be open, handling, or closed"})
	case err != nil:
		h.log.Error("chat set status", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": body.Status})
	}
}

// agentReply posts a human agent's reply into a conversation (marking it
// "handling") and delivers it out over the messaging channel when applicable.
func (h *chatHandler) agentReply(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text required"})
		return
	}
	convID := r.PathValue("id")
	channel, externalUser, err := h.svc.AgentReply(r.Context(), orgFromCtx(r), convID, body.Text)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	case err != nil:
		h.log.Error("chat agent reply", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	// Deliver over the messaging channel; web-widget threads have no transport
	// (the visitor reads the reply on their next widget fetch).
	if channel == "whatsapp" || channel == "messenger" {
		if err := h.msg.SendOutbound(r.Context(), orgFromCtx(r), channel, externalUser, body.Text); err != nil {
			h.log.Error("deliver agent reply", "channel", channel, "err", err)
			writeJSON(w, http.StatusOK, map[string]any{"status": "stored", "delivered": false})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "delivered": true})
}

// summarize (re)generates the AI insight for a conversation and returns it.
func (h *chatHandler) summarize(w http.ResponseWriter, r *http.Request) {
	in, err := h.svc.Summarize(r.Context(), orgFromCtx(r), r.PathValue("id"))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case err != nil:
		h.log.Error("chat summarize", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusOK, insightToDTO(in))
	}
}
