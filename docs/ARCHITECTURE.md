# Architecture

## Why it is built this way

The task is small enough that a single `main.go` with handlers talking straight to GORM would have
worked. I deliberately did not do that. The interesting part of this brief is concurrency and
correctness (no overselling, idempotent payments, non-blocking notifications), and those concerns are
much easier to reason about and test when the business rules live in one place that has no dependency
on the web framework or the database driver.

So the project is a hexagonal (ports and adapters) application with a domain-driven core. The rule I
held to throughout is that source dependencies only ever point inward, toward the domain.

```
        interfaces/http              (driving adapters: Gin handlers, middleware, WebSocket)
              |
              v
   +----------------------+
   |     application      |          use cases (command / query), event handlers,
   |  ports (interfaces)  |          and the PORTS that describe what they need
   +----------+-----------+
              |
              v
           domain                    aggregates, value objects, invariants, domain events.
                                     Pure Go. No gorm, no gin, nothing external.
              ^
              |
   +----------+-----------+
   |    infrastructure    |          driven adapters that IMPLEMENT the ports:
   |                      |          gorm repos, payment gateway, event bus, dispatcher, auth
   +----------------------+
```

The domain knows nothing about the layers around it. The application layer defines interfaces
(`ports`) for everything it needs from the outside (repositories, a unit of work, a payment gateway,
an event bus, a clock, password hashing, token issuing). Infrastructure provides concrete
implementations. The HTTP layer translates requests into use-case calls and use-case results back
into JSON. `internal/bootstrap` is the one place that knows about all of them and wires them
together.

## The layers

**Domain** (`internal/domain`). Seven aggregates: user, product, inventory, order, payment,
notification, audit, plus a small shared kernel (a `Money` value object and an event-recording
aggregate root). Entities have unexported fields and expose behaviour through methods that enforce
invariants. Each aggregate has two constructors: `New...` for genuinely new instances (validates and
records a creation event) and `Reconstitute...` for rebuilding from the database (no validation, no
events). State changes record domain events rather than mutating freely. The order aggregate is the
clearest example: it is a strict state machine (pending, reserved, paid, fulfilled, cancelled,
failed) and an illegal transition returns a typed error instead of corrupting state.

**Application** (`internal/application`). Use cases split into `command` (writes) and `query` (reads),
which keeps the order-placement path separate from reporting and listing. The ports live here because
they belong to the code that consumes them, not to the code that implements them. Event handlers
(notifications and audit) also live here and react to domain events.

**Infrastructure** (`internal/infrastructure`). The adapters. GORM repositories with persistence
models that are deliberately separate from the domain entities and mapped back and forth, so the
domain never carries `gorm` tags. A mock payment gateway. An in-process event bus. A notification
dispatcher. bcrypt and JWT. A system clock.

**Interfaces** (`internal/interfaces/http`). Gin handlers, request/response DTOs, middleware (request
ID, structured logging, recovery, metrics, CORS, JWT auth, role checks, rate limiting), the router,
and the WebSocket hub. Errors are mapped to HTTP status codes in one place based on a typed error
kind, so handlers stay thin.

**Platform** (`internal/platform`). Configuration loaded from the environment, and the HTTP server
with its graceful-shutdown logic.

## The order pipeline

Placing an order is the core flow and the place where most of the design decisions show up.

1. Idempotency check. If an order already exists for the request's idempotency key, return it and
   stop. Re-sending the same request never creates a second order or a second charge.
2. Build the line items, snapshotting each product's price at the moment of the order so later price
   changes do not rewrite history.
3. Submit the work to a bounded worker pool and wait for the result. The pool is what lets the system
   absorb a thousand simultaneous requests without opening a thousand database transactions at once.
4. Inside the pool, processing runs as two database transactions with the payment call in between:
   - Transaction one creates the order and reserves stock for every item. If any reservation fails
     the whole transaction rolls back, so there are never partial reservations and no orphaned order.
   - The payment call happens outside any transaction. This is on purpose. Holding a database
     transaction open across a network call to a payment provider is how you turn a slow third party
     into database lock contention.
   - Transaction two finalises the result. On success it records the payment, commits the reserved
     stock, and moves the order to fulfilled. On a decline it releases the reserved stock and marks
     the order failed.
5. The domain events recorded along the way are published to the event bus after the work completes.

## Concurrency model

The brief lists four concurrency scenarios. Here is how each one is handled.

**No overselling.** Inventory is not read, decremented in Go, then written back. That read-modify-write
is exactly the race that causes overselling. Instead, reservation is a single conditional update:

```sql
UPDATE inventories SET available = available - $qty, reserved = reserved + $qty, version = version + 1
WHERE product_id = $id AND available >= $qty
```

The database evaluates the `available >= $qty` condition and the decrement atomically. If the row was
updated, the reservation succeeded; if zero rows were affected, there was not enough stock. Two
buyers racing for the last unit serialise on that row and exactly one of them gets a row count of one.
`test/concurrency/inventory_reserve_test.go` proves this with 500 goroutines under the race detector.

**High-volume processing.** Order placement goes through a worker pool (`pkg/concurrency`) sized from
configuration. Requests queue and are processed with bounded parallelism, which gives backpressure
instead of resource exhaustion. The throughput test in `test/concurrency` places a thousand concurrent
orders and confirms the final stock never goes negative.

**Idempotent payments.** The payment gateway keys results by idempotency key, so a retried charge
returns the original result instead of charging again. The order's own idempotency key short-circuits
the whole pipeline before any work happens. Together these mean a client can safely retry.

**Non-blocking notifications.** When an order is paid or fails, the order pipeline does not send an
email. It publishes a domain event. The event bus dispatches to handlers on its own worker pool, and
the notification handler hands off to a dispatcher backed by a buffered channel. Nothing in the
critical path waits on a notification being sent. Audit logging works the same way through a wildcard
handler that observes every event.

## Events

The event bus is in-process by default. Handlers subscribe by event name (with `"*"` as a wildcard
for the audit handler). Publishing is asynchronous and the workers recover from panics so a faulty
handler cannot take down processing. If `RABBITMQ_ENABLED` is set, the in-process bus is wrapped in a
composite that also publishes each event to a RabbitMQ exchange, with broker errors logged and
swallowed so a message-queue problem never breaks order processing.

## Persistence and transactions

Repositories are defined as interfaces in the application ports and implemented with GORM in
infrastructure. Crossing a transaction boundary is expressed through a unit of work:

```go
uow.Do(ctx, func(repos ports.RepositoryProvider) error {
    // every repository obtained from repos runs on the same transaction
})
```

The `Do` method runs the function inside a GORM transaction and hands it a repository provider whose
repositories are all bound to that transaction. The application layer expresses "these operations are
atomic" without ever importing `gorm` or writing `BEGIN`/`COMMIT`.

## Cross-cutting decisions worth calling out

- Money is stored and moved around as an integer number of minor units (cents) inside a `Money` value
  object. No floats anywhere near prices.
- IDs are database-assigned. After an insert the repository copies the generated key back onto the
  aggregate so the rest of the flow and the response have a real ID to work with.
- Configuration comes entirely from the environment. No secret values are committed; `make env`
  generates local development secrets into a gitignored `.env`.
- The server drains on shutdown: stop accepting requests, then stop the worker pool, the event bus
  and the dispatcher in an order that lets in-flight events finish before their channels close.

## Trade-offs

A separate set of persistence models and mappers is more code than putting `gorm` tags on the domain
structs. I think the isolation is worth it here because it is what keeps the domain testable in
isolation and framework-free, which is the whole point of the exercise. For a smaller or more
CRUD-shaped service I would not necessarily pay that cost.
