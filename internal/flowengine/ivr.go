// Package flowengine executes flows. The IVR runtime is a pure state machine
// over domain.IVRSpec — no telephony imports — so it is fully unit-testable.
// The conversation runtime (VA/chat) is added later against the same idea.
package flowengine

import "github.com/ebnsina/yaver-api/internal/domain"

// IVR is the deterministic keypad runtime.
type IVR struct{}

func NewIVR() *IVR { return &IVR{} }

// Start positions the flow at its entry node and returns the first directive.
func (e *IVR) Start(f domain.Flow) (domain.Directive, *domain.FlowState, *domain.Outcome) {
	st := &domain.FlowState{Node: f.IVR.Entry}
	d, o := e.emit(f, st)
	return d, st, o
}

// OnDTMF applies a keypress and returns the next directive, or a terminal
// outcome when the flow ends.
func (e *IVR) OnDTMF(f domain.Flow, st *domain.FlowState, digit string) (domain.Directive, *domain.Outcome) {
	node := f.IVR.Nodes[st.Node]
	next, ok := node.On[digit]
	if !ok {
		next, ok = node.On["invalid"]
	}
	if !ok {
		return domain.Directive{Kind: "hangup"}, &domain.Outcome{Result: "no_answer", Status: domain.StatusNoAnswer}
	}
	st.Node = next
	return e.emit(f, st)
}

// OnTimeout handles a gather timeout (no keypress in time).
func (e *IVR) OnTimeout(f domain.Flow, st *domain.FlowState) (domain.Directive, *domain.Outcome) {
	node := f.IVR.Nodes[st.Node]
	next, ok := node.On["timeout"]
	if !ok {
		return domain.Directive{Kind: "hangup"}, &domain.Outcome{Result: "no_answer", Status: domain.StatusNoAnswer}
	}
	st.Node = next
	return e.emit(f, st)
}

// emit returns the directive for the current node, or a terminal outcome if the
// node ends the flow.
func (e *IVR) emit(f domain.Flow, st *domain.FlowState) (domain.Directive, *domain.Outcome) {
	node := f.IVR.Nodes[st.Node]
	if node.End || node.Result != "" {
		return domain.Directive{Kind: "hangup", Prompt: node.Say},
			&domain.Outcome{Result: node.Result, Status: domain.StatusCompleted}
	}
	if node.Gather != nil {
		return domain.Directive{Kind: "say_gather", Prompt: node.Say, Gather: node.Gather}, nil
	}
	return domain.Directive{Kind: "say", Prompt: node.Say}, nil
}
