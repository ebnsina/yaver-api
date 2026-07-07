// Package resend is a domain.EmailSender backed by the Resend API
// (https://resend.com). Provider-agnostic seam: swap for SES/Postmark/etc. by
// writing another adapter behind domain.EmailSender.
package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const endpoint = "https://api.resend.com/emails"

type Sender struct {
	apiKey string
	from   string
	http   *http.Client
}

func New(apiKey, from string) *Sender {
	return &Sender{apiKey: apiKey, from: from, http: &http.Client{Timeout: 10 * time.Second}}
}

func (s *Sender) Send(ctx context.Context, to, subject, body string) error {
	payload, err := json.Marshal(map[string]any{
		"from":    s.from,
		"to":      []string{to},
		"subject": subject,
		"text":    body,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return fmt.Errorf("resend: status %d: %s", resp.StatusCode, bytes.TrimSpace(msg))
	}
	return nil
}
