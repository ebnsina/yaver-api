// Package twilio implements domain.SMSSender against Twilio's Messages API.
package twilio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

var _ domain.SMSSender = (*Sender)(nil)

type Sender struct {
	client *http.Client
	base   string // Twilio API base; overridable in tests
	sid    string
	token  string
	from   string
}

// New builds the sender. from is the Twilio phone number (E.164) messages come
// from; sid/token are the account credentials.
func New(sid, token, from string) *Sender {
	return &Sender{
		client: &http.Client{Timeout: 15 * time.Second},
		base:   "https://api.twilio.com",
		sid:    sid,
		token:  token,
		from:   from,
	}
}

func (s *Sender) Send(ctx context.Context, to, text string) error {
	form := url.Values{"To": {to}, "From": {s.from}, "Body": {text}}
	endpoint := s.base + "/2010-04-01/Accounts/" + s.sid + "/Messages.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.sid, s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("twilio %d: %s", resp.StatusCode, msg)
	}
	return nil
}
