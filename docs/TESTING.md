# Testing the system

This guide covers both ways of checking the system: driving the API by hand, and running the
automated suites. The only prerequisite is Docker and Make. There is no need for a local Go install,
a local Postgres, or anything else, since the toolchain runs in containers.

## Start it up

```bash
make up      # builds the image, starts Postgres and the app, runs migrations on boot
make seed    # loads demo data (safe to run more than once)
```

Check it is alive:

```bash
make smoke
# or:
curl localhost:8080/health     # {"status":"ok",...}
curl localhost:8080/ready      # {"status":"ready"}  (this one pings Postgres)
```

The Swagger UI is at http://localhost:8080/docs and the raw spec at
http://localhost:8080/openapi.yaml.

## What the seeder gives you

`make seed` creates four users and ten products. The products are stocked deliberately so that every
scenario has data without any setup:

| Login | Password | Role |
|-------|----------|------|
| admin@ex.com | adminpass123 | admin |
| cathy@ex.com / dave@ex.com / erin@ex.com | custpass123 | customer |

Products of note:

- Product 1 (Mechanical Keyboard), stock 50: ordinary orders
- Product 5 (Laptop Stand), stock 0: out-of-stock behaviour (409)
- Product 8 (ANC Headphones), stock 1: the last-item / contention case
- Products 4, 5, 6 and 8 start at or below their reorder level, so the low-stock report has data the
  moment you boot

## Option A: Postman (easiest)

Import both files from `postman/` and select the "Easy Orders - Local" environment:

- `postman/easy-orders.postman_collection.json`
- `postman/easy-orders.postman_environment.json`

The collection is chained, so logging in stores the token, creating a product stores its id, and so
on. Run them top to bottom: Register Admin, Login Admin, Register Customer, Login Customer, Create
Product, Place Order, then the read and admin requests. See `postman/README.md` for details.

## Option B: curl walkthrough

```bash
B=http://localhost:8080/api/v1

# log in as the seeded admin and capture the token
ADMIN=$(curl -s -XPOST $B/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@ex.com","password":"adminpass123"}' | grep -oP '"token":"\K[^"]+')

# browse the catalogue (public)
curl -s "$B/products?page=1&size=20"

# log in as a seeded customer
CUST=$(curl -s -XPOST $B/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"cathy@ex.com","password":"custpass123"}' | grep -oP '"token":"\K[^"]+')

# place an order (product 1, quantity 2) -> comes back "fulfilled"
curl -s -XPOST $B/orders -H "Authorization: Bearer $CUST" -H 'Content-Type: application/json' \
  -d '{"items":[{"product_id":1,"quantity":2}]}'

# inspect it
curl -s $B/orders/1            -H "Authorization: Bearer $CUST"
curl -s $B/orders/1/status     -H "Authorization: Bearer $CUST"
curl -s $B/products/1/inventory -H "Authorization: Bearer $CUST"   # available dropped by 2

# admin views
curl -s $B/admin/orders              -H "Authorization: Bearer $ADMIN"
curl -s $B/admin/reports/daily       -H "Authorization: Bearer $ADMIN"
curl -s $B/admin/inventory/low-stock -H "Authorization: Bearer $ADMIN"
```

## Things worth poking at

- **Out of stock.** Order product 5 (stock 0), or order more of product 1 than exists. You get `409`
  with error code `order.out_of_stock`, and no order is created.
- **Idempotency.** Send the same order twice with the same `Idempotency-Key` header. The second call
  returns the same order instead of placing a new one or charging again.
- **Low stock.** Buy down product 1 until `available` drops to its reorder level, then call
  `/admin/inventory/low-stock` and watch it appear.
- **Auth and roles.** A request to `/orders` with no token returns `401`. A customer calling
  `POST /products` returns `403`.
- **Payment failures.** Set `PAYMENT_FAILURE_RATE=0.1` in `.env` and `make restart`, then place a
  handful of orders. Some will come back `failed`, and their reserved stock is released. The seeder
  sets this to 0 by default so the happy path is deterministic.
- **Live status.** The WebSocket endpoint is `/ws`; it pushes a small JSON message for each order
  event. A quick check that the upgrade works:
  ```bash
  curl -s -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" -H "Sec-WebSocket-Version: 13" \
    http://localhost:8080/ws | head -1     # HTTP/1.1 101 Switching Protocols
  ```

## Automated tests

```bash
make test               # unit tests: domain rules, the order pipeline, the worker pool
make test-integration   # integration + concurrency tests, under the race detector
make bench              # order-processing benchmark
```

Unit tests run with no database. The integration and concurrency suites spin up a throwaway
`orders_test` database (dropped and recreated each run) so they never disturb your working data.

What the suites are actually checking:

- `test/concurrency/inventory_reserve_test.go` is the overselling proof. It seeds a single unit, fires
  500 goroutines that all try to buy it, and asserts that exactly one succeeds and the final stock is
  zero. It runs under `-race`.
- `test/concurrency/order_throughput_test.go` places a thousand concurrent orders against a stocked
  product and asserts the stock never goes negative, which is the same guarantee at scale plus the
  worker pool under real load.
- `test/integration/api_test.go` drives the real HTTP stack end to end: register, log in, create a
  product, place an order, read it back, and the error cases (401, 403, 409, idempotency).
- The unit tests under `internal/application/command` cover the pipeline logic with in-memory fakes:
  the happy path, out of stock, a declined payment releasing stock, and idempotency.

## Optional: the full stack

`make up-all` additionally starts RabbitMQ (management UI on http://localhost:15672) and Prometheus
(http://localhost:9090). Set `RABBITMQ_ENABLED=true` to have the app publish events to the exchange.
