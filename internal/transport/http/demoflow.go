package http

import "github.com/ebnsina/yaver-api/internal/domain"

// demoOrderConfirmFlow is a hardcoded Phase 0 flow. Real flows come from the
// no-code builder → `flows` table in Phase 1.
func demoOrderConfirmFlow() domain.Flow {
	return domain.Flow{
		ID:      "flow_demo_order_confirm",
		OrgID:   "org_demo",
		Name:    "order_confirm",
		Version: 1,
		Channel: domain.ChannelVoice,
		Type:    domain.FlowIVR,
		Locale:  "bn",
		IVR: domain.IVRSpec{
			Entry: "greet",
			Nodes: map[string]domain.IVRNode{
				"greet": {
					Say:    &domain.Prompt{TTS: "আপনার {{order.total}} টাকার অর্ডারটি নিশ্চিত করতে ১ চাপুন।"},
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
