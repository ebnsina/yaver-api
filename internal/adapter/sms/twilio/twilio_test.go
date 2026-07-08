package twilio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendPostsToTwilioWithAuth(t *testing.T) {
	var gotPath, gotTo, gotFrom, gotBody, gotUser, gotPass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		u, p, _ := r.BasicAuth()
		gotUser, gotPass = u, p
		_ = r.ParseForm()
		gotTo, gotFrom, gotBody = r.FormValue("To"), r.FormValue("From"), r.FormValue("Body")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM1"}`))
	}))
	defer srv.Close()

	s := New("AC123", "tok", "+1555")
	s.base = srv.URL
	if err := s.Send(context.Background(), "+8801712345678", "your code is 4321"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/2010-04-01/Accounts/AC123/Messages.json" {
		t.Fatalf("path %q", gotPath)
	}
	if gotUser != "AC123" || gotPass != "tok" {
		t.Fatalf("basic auth %q/%q", gotUser, gotPass)
	}
	if gotTo != "+8801712345678" || gotFrom != "+1555" || gotBody != "your code is 4321" {
		t.Fatalf("form To=%q From=%q Body=%q", gotTo, gotFrom, gotBody)
	}
}

func TestSendErrorsOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	s := New("AC", "bad", "+1")
	s.base = srv.URL
	if err := s.Send(context.Background(), "+1", "x"); err == nil {
		t.Fatal("expected error on 401")
	}
}
