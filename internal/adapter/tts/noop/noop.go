// Package noop is a dev TTS that synthesizes nothing — the default when no
// speech provider is configured. Flows fall back to pre-recorded audio prompts.
package noop

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
)

var _ domain.TTS = TTS{}

type TTS struct{}

func New() *TTS { return &TTS{} }

func (TTS) Synthesize(context.Context, string, string) ([]byte, string, error) {
	return nil, "", nil
}
