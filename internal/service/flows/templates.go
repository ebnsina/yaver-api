package flows

import "github.com/ebnsina/yaver-api/internal/domain"

// Template is a starter flow the no-code builder can clone. The Name doubles as
// the ingest routing key (see service/ingest), so creating a template wires up
// the matching store event automatically.
type Template struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Channel     string `json:"channel"`
	Type        string `json:"type"`
	Locale      string `json:"locale"`
	Spec        []byte `json:"-"` // raw JSON, surfaced as SpecJSON in the DTO
}

// Templates returns the built-in starter flows, newest use-case last.
func Templates() []Template {
	return []Template{
		{
			Name:        "order_confirm",
			Title:       "COD order confirmation",
			Description: "Press 1 to confirm, 2 to cancel, 3 to reschedule. The v1 wedge.",
			Channel:     string(domain.ChannelVoice),
			Type:        string(domain.FlowIVR),
			Locale:      "en",
			Spec:        orderConfirmSpec,
		},
		{
			Name:        "cart_recovery",
			Title:       "Abandoned-cart nudge",
			Description: "Press 1 to hold the cart and get a callback, 2 to decline.",
			Channel:     string(domain.ChannelVoice),
			Type:        string(domain.FlowIVR),
			Locale:      "en",
			Spec:        cartRecoverySpec,
		},
		{
			Name:        "delivery_reminder",
			Title:       "Delivery reminder",
			Description: "Press 1 to confirm you're available, 2 to reschedule delivery.",
			Channel:     string(domain.ChannelVoice),
			Type:        string(domain.FlowIVR),
			Locale:      "en",
			Spec:        deliveryReminderSpec,
		},
	}
}

var orderConfirmSpec = []byte(`{
  "entry": "greet",
  "nodes": {
    "greet":      {"say": {"tts": "Press 1 to confirm your order, 2 to cancel, 3 to reschedule."},
                   "gather": {"digits": 1, "timeout_s": 6},
                   "on": {"1": "confirmed", "2": "cancelled", "3": "reschedule", "timeout": "no_input"}},
    "confirmed":  {"say": {"tts": "Thank you, your order is confirmed."}, "result": "confirmed",  "end": true},
    "cancelled":  {"say": {"tts": "Your order has been cancelled."},     "result": "cancelled",  "end": true},
    "reschedule": {"say": {"tts": "We'll call you to reschedule."},      "result": "reschedule", "end": true},
    "no_input":   {"result": "no_answer", "end": true}
  }
}`)

var cartRecoverySpec = []byte(`{
  "entry": "greet",
  "nodes": {
    "greet":    {"say": {"tts": "You left items in your cart. Press 1 to hold them and get a callback, 2 to decline."},
                 "gather": {"digits": 1, "timeout_s": 6},
                 "on": {"1": "held", "2": "declined", "timeout": "no_input"}},
    "held":     {"say": {"tts": "Great, we've held your cart and will call you back shortly."}, "result": "confirmed", "end": true},
    "declined": {"say": {"tts": "No problem, thanks for your time."},                            "result": "cancelled", "end": true},
    "no_input": {"result": "no_answer", "end": true}
  }
}`)

var deliveryReminderSpec = []byte(`{
  "entry": "greet",
  "nodes": {
    "greet":      {"say": {"tts": "Your order is out for delivery. Press 1 to confirm you're available, 2 to reschedule."},
                   "gather": {"digits": 1, "timeout_s": 6},
                   "on": {"1": "available", "2": "reschedule", "timeout": "no_input"}},
    "available":  {"say": {"tts": "Thank you, your delivery is confirmed."}, "result": "confirmed",  "end": true},
    "reschedule": {"say": {"tts": "We'll reschedule your delivery."},        "result": "reschedule", "end": true},
    "no_input":   {"result": "no_answer", "end": true}
  }
}`)
