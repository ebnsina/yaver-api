package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/internal/service/apikeys"
	"github.com/ebnsina/yaver-api/internal/service/ingest"
)

const orgKey ctxKey = 1

type ingestHandler struct {
	log    *slog.Logger
	keys   *apikeys.Service
	ingest *ingest.Service
}

// requireAPIKey authenticates X-API-Key and injects the org.
func (h *ingestHandler) requireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		org, ok, err := h.keys.Authenticate(r.Context(), r.Header.Get("X-API-Key"))
		if err != nil {
			h.log.Error("apikey auth", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
			return
		}
		ctx := context.WithValue(r.Context(), orgKey, org)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type eventBody struct {
	EventType string `json:"event_type"`
	EventID   string `json:"event_id"`
	Customer  struct {
		Phone      string `json:"phone"`
		Name       string `json:"name"`
		ExternalID string `json:"external_id"`
	} `json:"customer"`
}

func (h *ingestHandler) postEvent(w http.ResponseWriter, r *http.Request) {
	org, _ := r.Context().Value(orgKey).(domain.OrgID)

	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var body eventBody
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.EventType == "" || body.EventID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "event_type and event_id are required"})
		return
	}

	eventID, dup, err := h.ingest.Accept(r.Context(), org, ingest.Event{
		Type:         body.EventType,
		ExternalID:   body.EventID,
		Phone:        body.Customer.Phone,
		CustomerName: body.Customer.Name,
		CustomerRef:  body.Customer.ExternalID,
		Payload:      raw,
	})
	switch {
	case errors.Is(err, domain.ErrInvalidPhone):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid phone"})
	case err != nil:
		h.log.Error("ingest", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	case dup:
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
	default:
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted", "event_id": eventID})
	}
}

type apiKeyInfoDTO struct {
	Prefix     string  `json:"prefix"`
	Name       string  `json:"name"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt *string `json:"last_used_at"`
}

// listKeys returns the org's API keys (metadata only).
func (h *ingestHandler) listKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.keys.List(r.Context(), orgFromCtx(r))
	if err != nil {
		h.log.Error("list keys", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	out := make([]apiKeyInfoDTO, 0, len(keys))
	for _, k := range keys {
		var last *string
		if k.LastUsedAt != nil {
			s := k.LastUsedAt.Format(time.RFC3339)
			last = &s
		}
		out = append(out, apiKeyInfoDTO{Prefix: k.Prefix, Name: k.Name, CreatedAt: k.CreatedAt.Format(time.RFC3339), LastUsedAt: last})
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": out})
}

// mintKey creates an API key for the authenticated user's org and returns it once.
func (h *ingestHandler) mintKey(w http.ResponseWriter, r *http.Request) {
	full, err := h.keys.Mint(r.Context(), orgFromCtx(r), "dashboard")
	if err != nil {
		h.log.Error("mint key", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"api_key": full})
}
