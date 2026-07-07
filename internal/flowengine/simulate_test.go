package flowengine

import (
	"testing"

	"github.com/ebnsina/yaver-api/internal/domain"
)

func confirmFlow() domain.Flow {
	return domain.Flow{Type: domain.FlowIVR, IVR: domain.IVRSpec{
		Entry: "greet",
		Nodes: map[string]domain.IVRNode{
			"greet": {
				Say:    &domain.Prompt{TTS: "Press 1 to confirm, 2 to cancel."},
				Gather: &domain.Gather{Digits: 1, TimeoutS: 6},
				On:     map[string]string{"1": "confirmed", "2": "cancelled", "timeout": "no_input"},
			},
			"confirmed": {Result: "confirmed", End: true},
			"cancelled": {Result: "cancelled", End: true},
			"no_input":  {Result: "no_answer", End: true},
		},
	}}
}

func TestSimulateConfirmPath(t *testing.T) {
	sim := NewIVR().Simulate(confirmFlow(), []string{"1"})
	if !sim.Ended || sim.Result != "confirmed" {
		t.Fatalf("expected confirmed outcome, got ended=%v result=%q", sim.Ended, sim.Result)
	}
	if len(sim.Steps) != 2 {
		t.Fatalf("expected 2 steps (greet, confirmed), got %d", len(sim.Steps))
	}
	if sim.Steps[0].Node != "greet" || !sim.Steps[0].Awaits {
		t.Errorf("first step should be the gather at greet, got %+v", sim.Steps[0])
	}
	if sim.Steps[1].Node != "confirmed" || sim.Steps[1].Kind != "hangup" {
		t.Errorf("last step should hang up at confirmed, got %+v", sim.Steps[1])
	}
}

func TestSimulateTimeoutPath(t *testing.T) {
	sim := NewIVR().Simulate(confirmFlow(), []string{"timeout"})
	if !sim.Ended || sim.Result != "no_answer" {
		t.Errorf("timeout should reach no_answer, got ended=%v result=%q", sim.Ended, sim.Result)
	}
}

func TestSimulateRunsOutOfInputs(t *testing.T) {
	// No inputs: stops at the first gather, not yet ended.
	sim := NewIVR().Simulate(confirmFlow(), nil)
	if sim.Ended {
		t.Errorf("should not be ended while awaiting a keypress")
	}
	if len(sim.Steps) != 1 || !sim.Steps[0].Awaits {
		t.Errorf("expected a single awaiting step, got %+v", sim.Steps)
	}
}

func TestSimulateInvalidDigitHangsUp(t *testing.T) {
	// "9" has no transition and no "invalid" branch → no_answer hangup.
	sim := NewIVR().Simulate(confirmFlow(), []string{"9"})
	if !sim.Ended || sim.Status != string(domain.StatusNoAnswer) {
		t.Errorf("unmapped digit should end as no_answer, got ended=%v status=%q", sim.Ended, sim.Status)
	}
}
