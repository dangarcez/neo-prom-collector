APP_NAME := neo-collector
CMD := ./cmd/neo-collector
CONFIG ?= configs/config.demo.yaml
ENV_FILE ?= .env
GO ?= go
DOCKER_HOST ?= unix:///var/run/docker.sock
TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE ?= /var/run/docker.sock

.PHONY: build run run-once test test-integration

build:
	$(GO) build -o bin/$(APP_NAME) $(CMD)

run:
	$(GO) run $(CMD) -env $(ENV_FILE) -config $(CONFIG)

run-once:
	$(GO) run $(CMD) -env $(ENV_FILE) -config $(CONFIG) -once

test:
	$(GO) test ./...

test-integration:
	env -u XDG_RUNTIME_DIR \
	DOCKER_HOST=$(DOCKER_HOST) \
	TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=$(TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE) \
	$(GO) test -tags=integration -v ./tests/integration/...
