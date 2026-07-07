-- +goose Up
-- Phase 0 uses TEXT ids (matching the string ids the app generates) with no FK
-- constraints yet; FK/UUID alignment + partitioning come with real org/flow
-- creation in Phase 1.
CREATE TABLE flows (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    name       TEXT NOT NULL,
    version    INT NOT NULL DEFAULT 1,
    channel    TEXT NOT NULL,          -- voice | chat
    type       TEXT NOT NULL,          -- ivr | va | chat
    locale     TEXT NOT NULL,          -- language spoken to the customer
    spec       JSONB NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_flows_active ON flows (org_id, name) WHERE is_active;

CREATE TABLE calls (
    id               TEXT PRIMARY KEY,
    org_id           TEXT NOT NULL,
    flow_id          TEXT,
    provider_call_id TEXT,
    direction        TEXT NOT NULL,     -- inbound | outbound
    status           TEXT NOT NULL,     -- queued | ringing | ... | completed
    result           TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_calls_org_created ON calls (org_id, created_at DESC);

-- Seed the demo order-confirmation flow used by the /v1/dev endpoints.
INSERT INTO flows (id, org_id, name, version, channel, type, locale, spec) VALUES (
    'flow_demo_order_confirm', 'org_demo', 'order_confirm', 1, 'voice', 'ivr', 'bn',
    $json${
        "entry": "greet",
        "nodes": {
            "greet":      {"say": {"tts": "আপনার {{order.total}} টাকার অর্ডারটি নিশ্চিত করতে ১ চাপুন।"},
                           "gather": {"digits": 1, "timeout_s": 6},
                           "on": {"1": "confirmed", "2": "cancelled", "3": "reschedule", "timeout": "no_input"}},
            "confirmed":  {"say": {"audio": "confirmed.wav"},  "result": "confirmed",  "end": true},
            "cancelled":  {"say": {"audio": "cancelled.wav"},  "result": "cancelled",  "end": true},
            "reschedule": {"say": {"audio": "reschedule.wav"}, "result": "reschedule", "end": true},
            "no_input":   {"result": "no_answer", "end": true}
        }
    }$json$::jsonb
);

-- +goose Down
DROP TABLE IF EXISTS calls;
DROP TABLE IF EXISTS flows;
