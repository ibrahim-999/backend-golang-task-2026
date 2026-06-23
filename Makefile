.DEFAULT_GOAL := help
SHELL := /bin/bash

GO_IMAGE     ?= golang:1.25
COMPOSE      ?= docker compose
PROJECT      := order-processing
NETWORK      := $(PROJECT)_appnet
PWD_DIR      := $(shell pwd)
GOMOD_VOL    := opt-gomodcache
GOCACHE_VOL  := opt-gobuildcache

GO_RUN = docker run --rm -v $(PWD_DIR):/src -w /src -v $(GOMOD_VOL):/go/pkg/mod -v $(GOCACHE_VOL):/root/.cache/go-build -e CGO_ENABLED=0 $(GO_IMAGE)
GO_RUN_DB = docker run --rm --network $(NETWORK) --env-file .env -e DB_HOST=postgres -e DB_PORT=5432 -e DB_SSLMODE=disable -v $(PWD_DIR):/src -w /src -v $(GOMOD_VOL):/go/pkg/mod -v $(GOCACHE_VOL):/root/.cache/go-build -e CGO_ENABLED=1 $(GO_IMAGE)
GO_RUN_TESTDB = docker run --rm --network $(NETWORK) --env-file .env -e DB_HOST=postgres -e DB_PORT=5432 -e DB_SSLMODE=disable -e DB_NAME=orders_test -v $(PWD_DIR):/src -w /src -v $(GOMOD_VOL):/go/pkg/mod -v $(GOCACHE_VOL):/root/.cache/go-build -e CGO_ENABLED=1 $(GO_IMAGE)

.PHONY: help
help:
	@echo "Concurrent Order Processing System - Make targets"
	@echo ""
	@echo "  Setup"
	@echo "    make env             Copy .env.example to .env"
	@echo "    make tidy            go mod tidy (in the Go toolchain container)"
	@echo ""
	@echo "  Build & run (Docker)"
	@echo "    make build           Build the application image"
	@echo "    make up              Start postgres + app"
	@echo "    make up-all          Start everything (rabbitmq + prometheus profiles)"
	@echo "    make down            Stop the stack"
	@echo "    make down-v          Stop the stack and drop volumes"
	@echo "    make logs            Tail application logs"
	@echo "    make ps              Show running services"
	@echo "    make restart         Rebuild and restart the app service"
	@echo "    make smoke           Curl the health/ready/metrics endpoints"
	@echo ""
	@echo "  Quality"
	@echo "    make fmt             gofmt -w"
	@echo "    make vet             go vet"
	@echo "    make lint            golangci-lint"
	@echo "    make test            Unit tests"
	@echo "    make test-race       Unit tests with the race detector"
	@echo "    make test-integration  Integration + concurrency tests (needs postgres)"
	@echo "    make cover           Coverage report"
	@echo "    make bench           Benchmarks"
	@echo ""
	@echo "  Data"
	@echo "    make migrate         Run schema migrations"
	@echo "    make seed            Seed demo data"

.PHONY: env
env:
	@test -f .env || cp .env.example .env
	@grep -qE '^DB_PASSWORD=.+' .env || sed -i "s|^DB_PASSWORD=.*|DB_PASSWORD=$$(openssl rand -hex 16)|" .env
	@grep -qE '^RABBITMQ_PASSWORD=.+' .env || sed -i "s|^RABBITMQ_PASSWORD=.*|RABBITMQ_PASSWORD=$$(openssl rand -hex 16)|" .env
	@grep -qE '^JWT_SECRET=.+' .env || sed -i "s|^JWT_SECRET=.*|JWT_SECRET=$$(openssl rand -hex 32)|" .env
	@echo ".env ready (random local-only secrets generated; .env is gitignored)"

.PHONY: tidy
tidy:
	$(GO_RUN) sh -c "go mod tidy"

.PHONY: fmt
fmt:
	$(GO_RUN) gofmt -w cmd internal pkg

.PHONY: vet
vet:
	$(GO_RUN) go vet ./...

.PHONY: lint
lint:
	docker run --rm -v $(PWD_DIR):/src -w /src -v $(GOMOD_VOL):/go/pkg/mod golangci/golangci-lint:latest golangci-lint run ./...

.PHONY: build
build:
	$(COMPOSE) build app

.PHONY: up
up: env
	$(COMPOSE) up -d --build postgres app
	@echo "app:        http://localhost:8080"
	@echo "health:     http://localhost:8080/health"
	@echo "metrics:    http://localhost:8080/metrics"

.PHONY: up-all
up-all: env
	$(COMPOSE) --profile messaging --profile observability up -d --build
	@echo "app:        http://localhost:8080"
	@echo "rabbitmq:   http://localhost:15672 (credentials from .env)"
	@echo "prometheus: http://localhost:9090"

.PHONY: down
down:
	$(COMPOSE) --profile messaging --profile observability down

.PHONY: down-v
down-v:
	$(COMPOSE) --profile messaging --profile observability down -v

.PHONY: logs
logs:
	$(COMPOSE) logs -f app

.PHONY: ps
ps:
	$(COMPOSE) ps

.PHONY: restart
restart:
	$(COMPOSE) up -d --build app

.PHONY: smoke
smoke:
	@echo "GET /health";  curl -fsS http://localhost:8080/health  && echo ""
	@echo "GET /ready";   curl -fsS http://localhost:8080/ready   && echo ""
	@echo "GET /api/v1/ping"; curl -fsS http://localhost:8080/api/v1/ping && echo ""
	@echo "GET /metrics (first lines)"; curl -fsS http://localhost:8080/metrics | head -n 5

.PHONY: test
test:
	$(GO_RUN) go test -count=1 ./...

.PHONY: test-race
test-race:
	docker run --rm -v $(PWD_DIR):/src -w /src -v $(GOMOD_VOL):/go/pkg/mod -v $(GOCACHE_VOL):/root/.cache/go-build -e CGO_ENABLED=1 $(GO_IMAGE) go test -race -count=1 ./...

.PHONY: test-integration
test-integration: env
	$(COMPOSE) up -d postgres
	$(COMPOSE) exec -T postgres psql -U postgres -c "DROP DATABASE IF EXISTS orders_test WITH (FORCE)" >/dev/null 2>&1 || true
	$(COMPOSE) exec -T postgres psql -U postgres -c "CREATE DATABASE orders_test" >/dev/null 2>&1 || true
	$(GO_RUN_TESTDB) go test -race -p 1 -count=1 -tags=integration ./test/...

.PHONY: cover
cover: env
	$(COMPOSE) up -d postgres
	$(COMPOSE) --profile messaging up -d rabbitmq
	@for i in $$(seq 1 30); do [ "$$(docker inspect -f '{{.State.Health.Status}}' order-processing-rabbitmq-1 2>/dev/null)" = healthy ] && break; sleep 2; done
	$(COMPOSE) exec -T postgres psql -U postgres -c "DROP DATABASE IF EXISTS orders_test WITH (FORCE)" >/dev/null 2>&1 || true
	$(COMPOSE) exec -T postgres psql -U postgres -c "CREATE DATABASE orders_test" >/dev/null 2>&1 || true
	$(GO_RUN_TESTDB) sh -c "go test -tags=integration -p 1 -coverpkg=./internal/...,./pkg/... -coverprofile=coverage.out -count=1 ./... && go tool cover -func=coverage.out | tail -n 1"

.PHONY: bench
bench: env
	$(COMPOSE) up -d postgres
	$(COMPOSE) exec -T postgres psql -U postgres -c "DROP DATABASE IF EXISTS orders_test WITH (FORCE)" >/dev/null 2>&1 || true
	$(COMPOSE) exec -T postgres psql -U postgres -c "CREATE DATABASE orders_test" >/dev/null 2>&1 || true
	$(GO_RUN_TESTDB) go test -tags=integration -p 1 -bench=. -benchmem -run=^$$ ./test/...

.PHONY: migrate
migrate:
	$(COMPOSE) exec app /app/server migrate || $(COMPOSE) run --rm app migrate

.PHONY: seed
seed: env
	$(COMPOSE) up -d postgres
	$(GO_RUN_DB) go run -buildvcs=false ./cmd/seed
