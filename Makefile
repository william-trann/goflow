GO ?= go

.PHONY: fmt test build run api worker

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

build:
	$(GO) build ./...

run:
	$(GO) run ./cmd/forgeq

api:
	$(GO) run ./cmd/forgeq-api

worker:
	$(GO) run ./cmd/forgeq-worker
