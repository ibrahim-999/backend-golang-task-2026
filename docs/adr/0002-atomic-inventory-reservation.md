# ADR 0002: Prevent overselling with a conditional UPDATE

Status: Accepted

## Context

Multiple customers can try to buy the last unit of a product at the same time. The naive approach,
reading the stock level, checking it in application code, then writing the decremented value back, has
a race between the read and the write that lets two requests both believe the item is available.

## Decision

Never read-modify-write stock in Go. Reserve with a single conditional statement and let the database
do the check and the decrement atomically:

```sql
UPDATE inventories SET available = available - $qty, reserved = reserved + $qty, version = version + 1
WHERE product_id = $id AND available >= $qty
```

A reservation succeeded if and only if one row was affected. Concurrent updates to the same row
serialise, so exactly one of two racing requests for the last unit gets a row count of one.

## Consequences

Overselling is impossible regardless of how many requests arrive at once, without table locks or a
serialisable isolation level, and with one round trip. It is proven by a test that fires 500 goroutines
at a single unit under the race detector and asserts a single winner. The same statement style is used
for release, commit and restock so the invariant (available and reserved never go negative) holds on
every path.
