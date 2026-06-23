# ADR 0001: Hexagonal architecture with an inward dependency rule

Status: Accepted

## Context

The brief is concurrency-heavy: no overselling, idempotent payments, asynchronous notifications,
concurrent reporting. The risky logic is the business logic, and business logic is easiest to get
right and to test when it does not depend on a web framework or a database driver.

## Decision

Structure the project as ports and adapters. The domain is pure Go. The application layer defines
interfaces (ports) for everything it needs from the outside and contains the use cases. Infrastructure
implements those ports with GORM, the payment gateway, the event bus, and so on. The HTTP layer is a
driving adapter. Source dependencies only point inward.

## Consequences

The domain and the order pipeline can be unit-tested with in-memory fakes and no database. Swapping an
adapter (for example the in-process event bus for RabbitMQ) does not touch the domain or the use
cases. The cost is more types and explicit mapping between layers, which is more code than a direct
handler-to-GORM approach. For this problem the separation pays for itself.
