# Postman - Easy Orders API

Import these two files into Postman:

- `easy-orders.postman_collection.json` - all 25 requests in 6 folders (System, Auth, Users, Products, Orders, Admin)
- `easy-orders.postman_environment.json` - the **Easy Orders - Local** environment (base URL, credentials, captured tokens/ids)

After importing, pick **Easy Orders - Local** from the environment dropdown (top right).

## Prerequisite

The stack must be running:

```bash
make up        # http://localhost:8080
```

## Run order (the collection is chained)

Requests capture values into the environment automatically, so run them roughly top-to-bottom:

1. **Auth -> Register Admin**, then **Login Admin** - captures `admin_token` (+ `user_id`)
2. **Auth -> Register Customer**, then **Login Customer** - captures `customer_token`
3. **Products -> Create Product (admin)** - captures `product_id`
4. **Orders -> Place Order (customer)** - captures `order_id`; should return `status: fulfilled`
5. **Orders / Admin** - inspect the order, inventory, daily report, low-stock

Authenticated requests already send `Authorization: Bearer {{admin_token}}` or `{{customer_token}}`.

## Things to try

- **Idempotency** - send *Place Order* twice with the same `idempotency_key`; the second call returns the same order instead of creating a new one.
- **Payment failures** - set `PAYMENT_FAILURE_RATE=0.1` in `.env`, `make restart`, then place several orders; some land in `failed` (their reservation is released).
- **Low stock** - keep placing orders until `available <= reorder_level`, then run **Admin -> Low Stock Alerts**.
- **RBAC** - call **Products -> Create Product** with the customer token (swap the header to `{{customer_token}}`) and expect `403`.

## Run from the CLI (optional)

With [newman](https://github.com/postmanlabs/newman):

```bash
newman run easy-orders.postman_collection.json -e easy-orders.postman_environment.json
```
