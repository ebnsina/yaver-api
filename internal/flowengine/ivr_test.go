package flowengine

import (
	"testing"

	"github.com/ebnsina/yaver-api/internal/domain"
)

// orderConfirmFlow is the wedge flow: press 1 confirm / 2 cancel / 3 reschedule.
func orderConfirmFlow() domain.Flow {
	return domain.Flow{
		Type:   domain.FlowIVR,
		Locale: "bn",
		IVR: domain.IVRSpec{
			Entry: "greet",
			Nodes: map[string]domain.IVRNode{
				"greet": {
					Say:    &domain.Prompt{TTS: "আপনার {{order.total}} টাকার অর্ডার..."},
					Gather: &domain.Gather{Digits: 1, TimeoutS: 6},
					On:     map[string]string{"1": "confirmed", "2": "cancelled", "3": "reschedule", "timeout": "no_input"},
				},
				"confirmed":  {Say: &domain.Prompt{Audio: "confirmed.wav"}, Result: "confirmed", End: true},
				"cancelled":  {Say: &domain.Prompt{Audio: "cancelled.wav"}, Result: "cancelled", End: true},
				"reschedule": {Say: &domain.Prompt{Audio: "reschedule.wav"}, Result: "reschedule", End: true},
				"no_input":   {Result: "no_answer", End: true},
			},
		},
	}
}

func TestIVR_KeypressOutcomes(t *testing.T) {
	cases := []struct {
		digit      string
		wantResult string
	}{
		{"1", "confirmed"},
		{"2", "cancelled"},
		{"3", "reschedule"},
		{"9", "no_answer"}, // unmapped digit, no "invalid" branch -> terminal
	}
	e := NewIVR()
	for _, tc := range cases {
		t.Run(tc.digit, func(t *testing.T) {
			f := orderConfirmFlow()
			d, st, o := e.Start(f)
			if d.Kind != "say_gather" || o != nil {
				t.Fatalf("start: got kind=%q outcome=%v, want say_gather/nil", d.Kind, o)
			}
			_, out := e.OnDTMF(f, st, tc.digit)
			if out == nil {
				t.Fatalf("digit %s: expected terminal outcome, got nil", tc.digit)
			}
			if out.Result != tc.wantResult {
				t.Fatalf("digit %s: got %q, want %q", tc.digit, out.Result, tc.wantResult)
			}
		})
	}
}

func TestIVR_Timeout(t *testing.T) {
	e := NewIVR()
	f := orderConfirmFlow()
	_, st, _ := e.Start(f)
	_, out := e.OnTimeout(f, st)
	if out == nil || out.Result != "no_answer" {
		t.Fatalf("timeout: got %v, want no_answer", out)
	}
}
