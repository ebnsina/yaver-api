// Package whisper implements domain.STT against a Whisper-compatible
// transcription API (OpenAI /v1/audio/transcriptions and API-compatible
// self-hosted servers). Provider-agnostic behind the STT port.
package whisper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

var _ domain.STT = (*STT)(nil)

type STT struct {
	client *http.Client
	url    string // base, e.g. https://api.openai.com/v1
	key    string
	model  string
}

// New builds the adapter. model defaults to "whisper-1" when empty.
func New(baseURL, apiKey, model string) *STT {
	if model == "" {
		model = "whisper-1"
	}
	return &STT{
		client: &http.Client{Timeout: 60 * time.Second},
		url:    baseURL,
		key:    apiKey,
		model:  model,
	}
}

// Transcribe uploads the audio and returns its transcript. locale (e.g. "bn",
// "en") is passed as the language hint to improve multilingual accuracy.
func (s *STT) Transcribe(ctx context.Context, audio []byte, locale string) (string, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "call.wav")
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(audio); err != nil {
		return "", err
	}
	_ = mw.WriteField("model", s.model)
	if locale != "" {
		_ = mw.WriteField("language", locale)
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url+"/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if s.key != "" {
		req.Header.Set("Authorization", "Bearer "+s.key)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("whisper %d: %s", resp.StatusCode, msg)
	}

	var out struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Text, nil
}
