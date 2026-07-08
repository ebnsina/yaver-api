// Package noop is a dev STT that transcribes nothing — the default when no
// transcription provider is configured.
package noop

import (
	"context"

	"github.com/ebnsina/yaver-api/internal/domain"
)

var _ domain.STT = STT{}

type STT struct{}

func New() *STT { return &STT{} }

func (STT) Transcribe(context.Context, []byte, string) (string, error) { return "", nil }
