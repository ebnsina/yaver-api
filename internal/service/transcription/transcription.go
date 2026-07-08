// Package transcription turns a call recording into a stored transcript: it
// fetches the recording, runs it through the STT port, and attaches the text to
// the call. It's the consumer of the domain.STT seam.
package transcription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type Service struct {
	stt   domain.STT
	calls domain.CallRepo
	http  *http.Client
}

func New(stt domain.STT, calls domain.CallRepo) *Service {
	return &Service{stt: stt, calls: calls, http: &http.Client{Timeout: 60 * time.Second}}
}

// Transcribe downloads the recording at recordingURL, transcribes it in the
// given locale, and stores the transcript (and URL) on the call.
func (s *Service) Transcribe(ctx context.Context, callID domain.CallID, recordingURL, locale string) error {
	audio, err := s.download(ctx, recordingURL)
	if err != nil {
		return err
	}
	text, err := s.stt.Transcribe(ctx, audio, locale)
	if err != nil {
		return err
	}
	return s.calls.AttachMedia(ctx, callID, recordingURL, text)
}

func (s *Service) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch recording: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
