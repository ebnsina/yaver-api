# CLAUDE.md — yaver-api

Go backend for **Yaver** — a multi-channel customer-engagement platform (voice IVR/VA + AI chat + WhatsApp/Messenger) for e-commerce. This repo is the API + embedded workers. The SvelteKit frontend lives in the sibling repo **`yaver-web`**.

## Git & commits (IMPORTANT)
- Author every commit as **`ebnsina <ebnsina.me@gmail.com>`** — per-repo config:
  `git config user.name "ebnsina" && git config user.email "ebnsina.me@gmail.com"`.
- **Do NOT** add a `Co-Authored-By: Claude` trailer, and do NOT use any other identity.
- Remote uses the **`github-es`** SSH host alias
  (`git@github-es:ebnsina/yaver-api.git`).
- Internal planning docs under **`docs/`** (e.g. `docs/plan.md`) and `data/` are
  **gitignored / not committed** — keep the public repo clean (no plans/roadmap,
  no secrets). They exist locally for reference only.

## Architecture principles
- **Dependency inversion:** `internal/domain` is pure (no framework imports) and holds business types + all port interfaces (`Orchestrator`, `VoiceProvider`, `ChatTransport`, repos, `LLM`, `TTS`, `STT`, `Clock`). Adapters implement ports; services orchestrate; transport is thin. Dependencies point inward.
- **Every external system is a port** — orchestration (Hatchet), telephony (LiveKit), chat transports, payments, AI providers are all interfaces; swapping one is a new adapter, not a refactor.
- **Typed SQL, no ORM** — `sqlc` generates Go from hand-written SQL; repos wrap it behind domain interfaces.
- **Idempotent handlers** — the orchestrator delivers at-least-once; every task handler is safe to re-run.
- **Provider-agnostic AI** — never hardwire a vendor; the LLM/TTS/STT provider is config.

## Layout
```
cmd/yaver        # single binary: HTTP server + embedded workers
internal/
  domain/        # pure core + ports.go
  service/       # use-cases per bounded context
  flowengine/    # IVR graph + conversation runtimes
  adapter/       # postgres, orchestrator, voice, chat, payments, email, llm, tts, stt
  transport/http # chi router, middleware, handlers, SSE
  platform/      # config, logging/otel, pgxpool, redis, idgen, clock
pkg/             # reusable, dependency-free: apikey, crypto, hmacsig, phone
migrations/      # goose SQL
```

## Commands
```
make build       # go build ./...
make run         # run the api binary
make test        # go test ./...
make lint        # go vet ./...
make up / down   # docker compose infra (postgres, hatchet, ...)
make migrate-up  # goose migrations
make sqlc        # regenerate typed queries
```

## Conventions
- Thin transport, thin adapters, fat services, pure domain.
- Typed sentinel errors in `domain`; mapped to HTTP codes in one place.
- Config is env-only, parsed once into a typed struct; no globals.
- Never log secrets (tokens, decrypted creds, raw API keys); API-key prefixes are OK.
