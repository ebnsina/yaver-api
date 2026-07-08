package transcription

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ebnsina/yaver-api/internal/domain"
)

type fakeSTT struct {
	gotAudio  []byte
	gotLocale string
}

func (f *fakeSTT) Transcribe(_ context.Context, audio []byte, locale string) (string, error) {
	f.gotAudio, f.gotLocale = audio, locale
	return "hello world", nil
}

type fakeCalls struct {
	id        domain.CallID
	url, text string
	attached  bool
}

func (f *fakeCalls) Create(context.Context, *domain.Call) error { return nil }
func (f *fakeCalls) Get(context.Context, domain.CallID) (*domain.Call, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeCalls) ListByOrg(context.Context, domain.OrgID, int) ([]domain.Call, error) {
	return nil, nil
}
func (f *fakeCalls) Summary(context.Context, domain.OrgID) (domain.CallSummary, error) {
	return domain.CallSummary{}, nil
}
func (f *fakeCalls) AttachMedia(_ context.Context, id domain.CallID, url, text string) error {
	f.id, f.url, f.text, f.attached = id, url, text, true
	return nil
}
func (f *fakeCalls) DeleteBefore(context.Context, time.Time) (int64, error) { return 0, nil }

func TestTranscribeDownloadsRunsSTTAndStores(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("AUDIOBYTES"))
	}))
	defer srv.Close()

	stt := &fakeSTT{}
	calls := &fakeCalls{}
	if err := New(stt, calls).Transcribe(context.Background(), "call_1", srv.URL, "bn"); err != nil {
		t.Fatal(err)
	}
	if string(stt.gotAudio) != "AUDIOBYTES" || stt.gotLocale != "bn" {
		t.Fatalf("STT got audio=%q locale=%q", stt.gotAudio, stt.gotLocale)
	}
	if !calls.attached || calls.id != "call_1" || calls.text != "hello world" || calls.url != srv.URL {
		t.Fatalf("AttachMedia not called correctly: %+v", calls)
	}
}
