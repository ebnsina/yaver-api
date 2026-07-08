package flowengine

import (
	"context"
	"testing"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// scriptedLeg answers gathers from a fixed list of keypresses (or times out when
// exhausted) and records what it played — a stand-in for a real media leg.
type scriptedLeg struct {
	digits []string
	i      int
	plays  int
	hungUp bool
}

func (l *scriptedLeg) Play(context.Context, *domain.Prompt) error { l.plays++; return nil }
func (l *scriptedLeg) SayGather(context.Context, *domain.Prompt, *domain.Gather) (string, bool, error) {
	l.plays++
	if l.i >= len(l.digits) {
		return "", true, nil // timeout
	}
	d := l.digits[l.i]
	l.i++
	return d, false, nil
}
func (l *scriptedLeg) Hangup(context.Context, *domain.Prompt) error { l.hungUp = true; return nil }

func TestRunSession_DrivesToOutcome(t *testing.T) {
	cases := []struct {
		digits []string
		want   string
	}{
		{[]string{"1"}, "confirmed"},
		{[]string{"2"}, "cancelled"},
		{nil, "no_answer"}, // no keypress → timeout branch
	}
	e := NewIVR()
	for _, tc := range cases {
		leg := &scriptedLeg{digits: tc.digits}
		out, err := e.RunSession(context.Background(), orderConfirmFlow(), leg)
		if err != nil {
			t.Fatal(err)
		}
		if out.Result != tc.want {
			t.Fatalf("digits %v: got %q, want %q", tc.digits, out.Result, tc.want)
		}
		if !leg.hungUp {
			t.Fatalf("digits %v: leg should be hung up at the end", tc.digits)
		}
	}
}
