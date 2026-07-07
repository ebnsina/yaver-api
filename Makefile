.PHONY: build run test lint tidy

# Load .env (gitignored) for local dev if present, so `make run` has the required
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
