.PHONY: build run test lint tidy sqlc up down migrate-up migrate-down migrate-status hatchet-up hatchet-down

# Load .env (gitignored) for local dev if present, so targets have the required
# env without hardcoded defaults. In prod, env is supplied by the platform.
ifneq (,$(wildcard ./.env))
include .env
export
endif

build:
	go build ./...

run:
	go run ./cmd/yaver

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

sqlc:
	sqlc generate

up:
	docker compose up -d

down:
	docker compose down

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

# Self-hosted Hatchet (lite) — the durable orchestrator. After `make hatchet-up`,
# create an API token in its dashboard (:8888), export HATCHET_CLIENT_TOKEN +
# HATCHET_CLIENT_TLS_STRATEGY=none, and run with YAVER_ORCHESTRATOR=hatchet.
hatchet-up:
	docker compose -f deploy/hatchet/docker-compose.yml up -d

hatchet-down:
	docker compose -f deploy/hatchet/docker-compose.yml down
