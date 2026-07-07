// Package meta is a MessagingSender that delivers replies via the Meta Graph
// API (WhatsApp Cloud API and Messenger Send API), first-party, no BSP.
package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

const graphBase = "https://graph.facebook.com/v21.0"

type Sender struct{ client *http.Client }

func New() *Sender { return &Sender{client: &http.Client{Timeout: 10 * time.Second}} }

func (s *Sender) Send(ctx context.Context, m domain.OutboundMessage) error {
	var url string
	var body any
	switch m.Type {
	case "whatsapp":
		url = fmt.Sprintf("%s/%s/messages?access_token=%s", graphBase, m.FromExternalID, m.AccessToken)
		body = map[string]any{
			"messaging_product": "whatsapp",
			"to":                m.To,
			"type":              "text",
			"text":              map[string]string{"body": m.Text},
		}
	case "messenger":
		url = fmt.Sprintf("%s/%s/messages?access_token=%s", graphBase, m.FromExternalID, m.AccessToken)
		body = map[string]any{
			"recipient": map[string]string{"id": m.To},
			"message":   map[string]string{"text": m.Text},
		}
	default:
		return fmt.Errorf("meta: unsupported channel %q", m.Type)
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("meta send: %s: %s", resp.Status, string(b))
	}
	return nil
}
