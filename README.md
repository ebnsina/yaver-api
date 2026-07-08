# Yaver API

Go backend for **Yaver** — a multi-channel customer-engagement platform for
e-commerce. Yaver reaches customers over **voice** (IVR / virtual agent),
**AI chat**, and **WhatsApp / Messenger**, and drives outcomes like order
confirmation, cart recovery, and delivery reminders. This repo is the HTTP API
plus embedded background workers. The SvelteKit dashboard lives in the sibling
repo **[`yaver-web`](../yaver-web)**.

## Highlights

- **Multi-channel engagement** — voice IVR/VA, an embeddable AI chat widget, and
  WhatsApp / Messenger via the Meta Graph API.
- **No-code flow builder** — IVR graphs and conversation runtimes with starter
  templates and an in-browser simulator (`/v1/flows/simulate`).
- **Event-driven automations** — merchants POST commerce events (`/v1/events`)
  that route into flows: cart-recovery, delivery-reminder, order-confirm.
- **Campaigns** — bulk outbound with CSV recipient import, scheduling, and a
  due-sweep worker.
- **Human takeover** — agents can take over an AI conversation; auto-reply is
  suppressed while a human is handling it.
- **Billing** — prepaid credits, low-balance email alerts, and top-ups via a
  gateway port (SSLCommerz in prod — cards + bKash/Nagad/Rocket; a mock gateway
  for dev). Credits are granted only on a gateway-authenticated IPN, idempotently.
- **Analytics & reports** — an overview endpoint plus a natural-language
  "ask" endpoint over the org's activity.
- **Abuse throttling** — per-IP token-bucket rate limiting on the OTP, ingest,
  and public-widget endpoints (in-process; swappable for a Redis limiter).
- **Real-time activity feed** — an org-scoped Server-Sent Events stream
  (`/v1/activity/stream`) pushes calls and chat messages to the dashboard live,
  fed by an in-process pub/sub bus behind a swappable port.
- **Tracing** — OpenTelemetry HTTP request spans, opt-in via an OTLP endpoint
  (standard `OTEL_*` env), exporting to any OTel backend (Grafana Tempo, Jaeger,
  Honeycomb, …).

## Architecture

The codebase follows **dependency inversion**: `internal/domain` is pure (no
framework imports) and owns the business types and every port interface
(`Orchestrator`, `VoiceProvider`, `ChatTransport`, repos, `LLM`, `TTS`, `STT`,
`Clock`). Adapters implement ports, services orchestrate use-cases, and the HTTP
transport stays thin. **Every external system is a port** — telephony (LiveKit),
orchestration (Hatchet), chat/messaging transports, payments, and AI providers
are all interfaces, so swapping one is a new adapter, not a refactor.

- **Typed SQL, no ORM** — `sqlc` generates Go from hand-written SQL; repos wrap
  it behind domain interfaces.
- **Idempotent handlers** — the orchestrator delivers at-least-once; every task
  handler is safe to re-run.
- **Provider-agnostic AI** — the LLM/TTS/STT provider is config, never hardwired.

### Layout

```
cmd/yaver        # single binary: HTTP server + embedded workers
cmd/migrate      # goose migration runner
internal/
  domain/        # pure core + ports.go
  service/       # use-cases per bounded context (auth, calls, flows, chat, …)
  flowengine/    # IVR graph + conversation runtimes
  adapter/       # postgres, orchestrator, voice, chat, messaging, email, reporter
  transport/http # net/http router, handlers, DTOs, SSE
  platform/      # config, db (pgxpool), clock
pkg/             # reusable, dependency-free: apikey, crypto, phone, id
migrations/      # goose SQL
```

## Getting started

### Prerequisites

- Go **1.26+**
- Docker (for local Postgres via `make up`)

### Run locally

```sh
cp .env.example .env          # fill in the required vars (see below)
make up                       # start Postgres
make migrate-up               # apply goose migrations
make run                      # start the API on :8080
```

Check it's alive:

```sh
curl localhost:8080/healthz          # {"status":"ok"}
curl localhost:8080/openapi.yaml     # served OpenAPI spec
```

The DB pool dials lazily, so the binary boots even without a live database —
handy for smoke tests.

## Configuration

Config is **env-only**, parsed once into a typed struct with no hardcoded
defaults — the app fails to boot if a required var is missing. Copy
`.env.example` to `.env` for local dev (`.env` is gitignored). Key vars:

| Var | Purpose |
| --- | --- |
| `YAVER_ENV` | `dev` / `prod` (controls secure cookies, etc.) |
| `YAVER_PORT` | HTTP listen port |
| `YAVER_DATABASE_URL` | Postgres DSN |
| `YAVER_AUTH_SECRET` | HMAC key for hashing OTP codes at rest |
| `YAVER_ENCRYPTION_KEY` | AES-256-GCM master key (base64 of 32 bytes) for secrets at rest — `openssl rand -base64 32` |
| `YAVER_ORCHESTRATOR` | `local` (in-process, no deps) or `hatchet` |
| `YAVER_CHAT_PROVIDER` | `builtin` (rule-based) or `anthropic` (Claude via the official SDK) |
| `YAVER_ANTHROPIC_API_KEY` | Required when `YAVER_CHAT_PROVIDER=anthropic` |
| `YAVER_ANTHROPIC_MODEL` | Optional; defaults to `claude-opus-4-8` |
| `YAVER_MSG_SENDER` | `log` (dev) or `meta` (WhatsApp/Messenger Graph API) |
| `YAVER_EMAIL_SENDER` | `log` (dev) or `resend` |
| `YAVER_EMAIL_FROM` | From address for transactional email |
| `YAVER_PAYMENT_GATEWAY` | `mock` (dev) or `sslcommerz` (cards + bKash/Nagad/Rocket) |
| `YAVER_APP_URL` | Public API base URL for gateway redirect/IPN callbacks |
| `YAVER_WEB_URL` | Dashboard base URL to return the customer to after paying |
| `YAVER_SSLCOMMERZ_STORE_ID` / `_STORE_PASSWD` / `_SANDBOX` | Required for the `sslcommerz` gateway |

For `hatchet`, also set `HATCHET_CLIENT_TOKEN` and
`HATCHET_CLIENT_TLS_STRATEGY`. For `resend`, set `YAVER_RESEND_API_KEY`. Never
log secrets — API-key prefixes are OK.

## API surface

The full contract is served at `GET /openapi.yaml`. Authentication modes:

- **Session cookie** (`/v1/*` dashboard routes) — phone-OTP login issues an
  httpOnly cookie; the org is resolved from the session.
- **`X-API-Key`** (`/v1/events`) — merchant server-to-server event ingest.
- **Publishable key** (`/public/*`, `/widget.js`) — cross-origin chat widget.
- **Meta signature** (`/webhooks/meta`) — inbound WhatsApp / Messenger.

Selected routes:

```
POST /v1/auth/otp/request | /v1/auth/otp/verify | /v1/auth/logout   # auth
GET  /v1/me                                                          # session
GET  /v1/calls | /v1/calls/{id}                                      # calls
GET/PUT /v1/settings/call-policy                                     # call policy
GET  /v1/flows | POST /v1/flows | GET /v1/flows/templates            # flows
POST /v1/flows/simulate                                              # flow simulator
GET/POST /v1/campaigns | POST /v1/campaigns/{id}/recipients|schedule|start
GET/POST /v1/chat/conversations | .../reply | .../status            # chat + takeover
GET/POST /v1/channels                                               # WhatsApp/Messenger
GET  /v1/billing | POST /v1/billing/topup | .../checkout           # credits + pay
POST /webhooks/payment                                             # gateway IPN
GET  /v1/analytics/overview | POST /v1/reports/ask                 # analytics
GET  /v1/activity/stream                                            # live SSE feed
POST /v1/events                                                     # merchant ingest
```

## Development

```sh
make build       # go build ./...
make run         # run the API binary
make test        # go test ./...
make lint        # go vet ./...
make sqlc        # regenerate typed queries from SQL
make up / down   # docker compose infra (Postgres)
make migrate-up  # goose migrations (also: migrate-down, migrate-status)
make hatchet-up  # run the self-hosted Hatchet engine (also: hatchet-down)
```

### Orchestrator (Hatchet)

Background jobs (e.g. `place_call`) run through `domain.Orchestrator`. The
default is the in-process `local` dispatcher (no deps). For durable, fairness-
keyed execution with retries and a dashboard, run the self-hosted Hatchet
engine and switch to it:

```sh
make hatchet-up          # engine + its own Postgres (dashboard :8888, gRPC :7077)
# create an API token in the dashboard, then:
export HATCHET_CLIENT_TOKEN=<token> HATCHET_CLIENT_TLS_STRATEGY=none
YAVER_ORCHESTRATOR=hatchet make run
```

The `place_call` task carries a per-merchant fairness key (`input.orgId`,
round-robin) so one merchant can't monopolize call slots.

Conventions: thin transport, thin adapters, fat services, pure domain. Typed
sentinel errors live in `domain` and map to HTTP status codes in one place.

## Deployment

The `Dockerfile` builds a static binary into a distroless image (the migration
runner is included):

```sh
docker build -t yaver-api .
# Apply migrations, then boot the API:
docker run --rm --env-file .env --entrypoint /app/migrate yaver-api up
docker run -p 8080:8080 --env-file .env yaver-api
```

Config is env-only (see the table above); supply it via the platform's secret
store in production. In front of the app, a reverse proxy terminates TLS and
routes the dashboard's `/v1`, `/public`, and `/widget.js` to this service.

## License

Proprietary — © Yaver. All rights reserved.
