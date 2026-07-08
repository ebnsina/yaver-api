package domain

import "context"

// TTS renders text to speech audio for a locale. Provider-agnostic seam — the
// voice media pipeline uses it to voice a flow's dynamic (TTS) prompts.
type TTS interface {
	Synthesize(ctx context.Context, text, locale string) (audio []byte, mime string, err error)
}

// STT transcribes call audio to text (multilingual, locale-hinted). Turns
// recordings into transcripts that feed the dashboard and AI reports.
type STT interface {
	Transcribe(ctx context.Context, audio []byte, locale string) (text string, err error)
}
