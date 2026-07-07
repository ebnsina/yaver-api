package flowengine

import "github.com/ebnsina/yaver-api/internal/domain"

// SimStep is one turn the IVR produced while simulating a flow.
type SimStep struct {
	Node   string         // node that produced this directive
	Kind   string         // "say" | "say_gather" | "hangup"
	Say    *domain.Prompt // prompt spoken at this step, if any
	Awaits bool           // true when the runtime is waiting for a keypress
}

// Simulation is the full trace of a dry-run plus its terminal outcome — the data
// behind the builder's in-browser simulator (no telephony, credits, or persistence).
type Simulation struct {
	Steps  []SimStep
	Ended  bool   // true once a terminal node/hangup was reached
	Result string // e.g. "confirmed" | "cancelled" (empty until Ended)
	Status string // domain.CallStatus
}

// Simulate runs the IVR over a sequence of inputs — each either a digit ("1".."9")
// or the literal "timeout" — recording every step. It stops at the first terminal
// outcome, or when inputs run out at a gather (Ended=false).
func (e *IVR) Simulate(f domain.Flow, inputs []string) Simulation {
	st := &domain.FlowState{Node: f.IVR.Entry}
	sim := Simulation{}

	d, o := e.emit(f, st)
	sim.Steps = append(sim.Steps, stepOf(st.Node, d))
	if o != nil {
		return finish(sim, o)
	}

	for _, in := range inputs {
		var out *domain.Outcome
		if in == "timeout" {
			d, out = e.OnTimeout(f, st)
		} else {
			d, out = e.OnDTMF(f, st, in)
		}
		sim.Steps = append(sim.Steps, stepOf(st.Node, d))
		if out != nil {
			return finish(sim, out)
		}
	}
	return sim // ran out of inputs while still awaiting a keypress
}

func stepOf(node string, d domain.Directive) SimStep {
	return SimStep{Node: node, Kind: d.Kind, Say: d.Prompt, Awaits: d.Kind == "say_gather"}
}

func finish(sim Simulation, o *domain.Outcome) Simulation {
	sim.Ended = true
	sim.Result = o.Result
	sim.Status = string(o.Status)
	return sim
}
