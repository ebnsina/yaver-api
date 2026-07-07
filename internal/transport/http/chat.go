package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/chat"
)

type chatHandler struct {
	log *slog.Logger
	svc *chat.Service
}

type conversationDTO struct {
	ID           string `json:"id"`
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
		out = append(out, conversationDTO{
			ID: c.ID, Status: c.Status, LastMessage: c.LastMessage,
			MessageCount: c.MessageCount, UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": out})
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
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}
