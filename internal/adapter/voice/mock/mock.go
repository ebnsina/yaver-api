// Package mock is a VoiceProvider that performs no telephony. It lets the whole
// flow → orchestration → outcome path run in tests and local dev with no telco.
package mock

import (
	"context"
	"log/slog"

	"github.com/ebnsina/yaver-api/internal/domain"
	"github.com/ebnsina/yaver-api/pkg/id"
)

type Provider struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Provider { return &Provider{log: log} }

func (p *Provider) PlaceCall(ctx context.Context, req domain.CallRequest) (domain.ProviderCallID, error) {
	pid := domain.ProviderCallID(id.New("mockcall"))
	p.log.Info("mock place_call",
		"provider_call_id", pid,
		"to", req.ToPhone,
		"flow", req.Flow.Name,
		"locale", req.Flow.Locale,
	)
	return pid, nil
}
