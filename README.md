# Concurrent Order Processing System

A backend service for high-volume e-commerce order processing, written in Go. It covers the full
order lifecycle (placement, inventory reservation, payment, fulfilment, notification and reporting)
and treats correctness under concurrency as the primary goal rather than an afterthought.

Built for the Easy Orders senior backend assessment.

## What it does

- JWT-authenticated REST API with role-based access (customer / admin)
- Order processing through a bounded worker pool, so a burst of traffic degrades gracefully instead
  of exhausting the database
- Race-free inventory. Overselling is prevented at the database level, and there is a test that fires
  500 goroutines at the last remaining unit and asserts exactly one of them wins
- Idempotent payments with bounded retry and backoff against a simulated gateway (configurable
  latency and failure rate)
- Event-driven notifications and audit logging that run asynchronously and never block an order
- Concurrent daily sales reporting
- Prometheus metrics, structured logging, connection pooling, graceful shutdown
- Live order-status updates over WebSocket and optional RabbitMQ event publishing
- OpenAPI spec served as Swagger UI, a Postman collection, and Kubernetes manifests

## Tech stack

Go 1.25, Gin, GORM v2, PostgreSQL 15. zerolog for logging, the Prometheus client for metrics,
golang-jwt for tokens, gorilla/websocket, and amqp091-go for RabbitMQ. Everything builds and runs in
Docker, so the only thing the host needs is Docker and Make.

## Running it

```bash
make up      # build the image, start Postgres and the app (migrations run on boot)
make seed    # load demo data: 4 users and 10 products
make smoke   # curl the health/readiness/metrics endpoints
```

The API listens on http://localhost:8080 and the Swagger UI is at http://localhost:8080/docs.

Logins created by the seeder:

| Email | Password | Role |
|-------|----------|------|
| admin@ex.com | adminpass123 | admin |
| cathy@ex.com | custpass123 | customer |

(`dave@ex.com` and `erin@ex.com` exist too, same customer password.)

For a full walkthrough, the Postman collection, and how to run the automated suites, see
[docs/TESTING.md](docs/TESTING.md).

## Architecture

The code is organised as a hexagonal (ports and adapters) application with a domain-driven core and
an event-driven processing model. The domain layer is plain Go and has no idea that Gin, GORM or
PostgreSQL exist. [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) goes through the layers, the
concurrency model and the decisions behind them.

The short version:

```
interfaces/http  ->  application (use cases + ports)  ->  domain
                              ^
                     infrastructure (gorm, payment, messaging, observability)

Every source dependency points inward. The domain is pure Go.
```

## Project layout

```
cmd/server              process entrypoint
cmd/seed                demo data loader
internal/domain         aggregates, value objects and domain events (no framework imports)
internal/application    use cases (command/query), event handlers, and the ports they depend on
internal/infrastructure adapters: gorm repositories, mock payment gateway, event bus, dispatcher, auth
internal/interfaces     HTTP delivery: handlers, middleware, DTOs, router, websocket
internal/bootstrap      composition root that wires everything together
internal/platform       configuration and HTTP server lifecycle
pkg                     small reusable pieces (worker pool, typed errors)
deploy/k8s              Kubernetes manifests
docs                    architecture notes, testing guide, OpenAPI spec
postman                 Postman collection and environment
test                    integration and concurrency tests
```

## Make targets

`make help` prints the full list. The ones used most:

```
make up / make down        start / stop the stack
make seed                  load demo data (idempotent)
make test                  unit tests
make test-integration      integration + concurrency tests under the race detector
make bench                 order-processing benchmark
make logs / ps / smoke
```

Integration tests and the benchmark run against a throwaway `orders_test` database that is dropped
and recreated each run, so they never touch the data you are working with.

## Database and migrations

The schema is managed with GORM's AutoMigrate, which runs on startup and again from the seeder, so
the eight tables are always present. I kept the column definitions and indexes on the GORM models
(`internal/infrastructure/persistence/gormrepo/models.go`) instead of maintaining a separate set of
hand-written SQL migration files.

## Notes on testing and scope

Statement coverage is about 66% when the integration suite is included (`make cover`). The domain
and the order pipeline are close to fully covered by unit tests; the GORM repositories and HTTP
handlers are covered by the integration suite, which drives the real server against Postgres. The
integration and concurrency suites run against a throwaway database that is dropped and recreated
each run, so they never touch the data you are working with.

Payments are idempotent and retried with bounded backoff (`PAYMENT_MAX_ATTEMPTS`,
`PAYMENT_RETRY_BACKOFF`). A transient decline is retried, and because the gateway de-duplicates on
the idempotency key a retry never double-charges. The order only fails, releasing its stock, once
the attempts are exhausted.

RabbitMQ is optional (`RABBITMQ_ENABLED`). When enabled, the in-process event bus is wrapped so every
domain event is also published to a topic exchange, with a consumer reading them back; broker errors
are logged and swallowed so a queue problem never breaks order processing.

One detail I would still tidy up: the first event an aggregate records (before the database has
assigned its ID) carries an aggregate id of 0. The persisted rows and every later lifecycle event
have the real IDs; only those initial event payloads are affected. Generating IDs in the application
rather than relying on the database would remove this.
