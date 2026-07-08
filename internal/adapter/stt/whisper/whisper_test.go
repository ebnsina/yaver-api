package whisper

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranscribeSendsMultipartAndParsesText(t *testing.T) {
	var gotModel, gotLang, gotAuth, gotCT string
	var gotFile bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		_ = r.ParseMultipartForm(1 << 20)
		gotModel = r.FormValue("model")
		gotLang = r.FormValue("language")
		if f, _, err := r.FormFile("file"); err == nil {
			b, _ := io.ReadAll(f)
			gotFile = string(b) == "AUDIO"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"order confirmed"}`))
	}))
	defer srv.Close()

	s := New(srv.URL, "sk-test", "")
	text, err := s.Transcribe(context.Background(), []byte("AUDIO"), "bn")
	if err != nil {
		t.Fatal(err)
	}
	if text != "order confirmed" {
		t.Fatalf("got transcript %q", text)
	}
	if gotModel != "whisper-1" {
		t.Fatalf("default model not sent, got %q", gotModel)
	}
	if gotLang != "bn" {
		t.Fatalf("language hint not sent, got %q", gotLang)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth header wrong: %q", gotAuth)
	}
	if !strings.HasPrefix(gotCT, "multipart/form-data") {
		t.Fatalf("not multipart: %q", gotCT)
	}
	if !gotFile {
		t.Fatal("audio file not uploaded")
	}
}

func TestTranscribeErrorsOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`bad key`))
	}))
	defer srv.Close()
	if _, err := New(srv.URL, "x", "").Transcribe(context.Background(), []byte("A"), ""); err == nil {
		t.Fatal("expected an error on 401")
	}
}
