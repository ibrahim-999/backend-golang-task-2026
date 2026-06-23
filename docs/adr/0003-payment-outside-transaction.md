# ADR 0003: Keep the payment call outside the database transaction

Status: Accepted

## Context

Placing an order touches the database (create the order, reserve stock, record the payment, fulfil)
and also calls an external payment provider. The simple version wraps all of it in one transaction.
The problem is that the payment call is a network request to a third party, and holding a database
transaction open across it turns the provider's latency into database lock contention. Under load that
is a fast way to exhaust the connection pool.

## Decision

Split processing into two transactions with the payment call in between.

- Transaction one: create the order and reserve stock. If any reservation fails the whole transaction
  rolls back, so there are no partial reservations and no orphaned order.
- Payment: called with no transaction open. The gateway is idempotent on the idempotency key.
- Transaction two: on success record the payment, commit the reserved stock and fulfil; on a decline
  release the stock and fail the order.

## Consequences

A slow or failing payment provider never holds database locks. The atomic reservation from ADR 0002 is
what makes this safe: stock is already protected the moment transaction one commits, so the gap before
transaction two cannot oversell. The tradeoff is that the flow is no longer a single transaction, so
the code has to handle the in-between state explicitly, which it does through the order state machine.
