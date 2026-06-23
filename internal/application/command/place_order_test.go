package command

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/audit"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/notification"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/payment"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/concurrency"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type fakeStore struct {
	mu          sync.Mutex
	users       map[uint64]*user.User
	usersEmail  map[string]uint64
	products    map[uint64]*product.Product
	orders      map[uint64]*order.Order
	ordersByKey map[string]uint64
	payments    map[uint64]*payment.Payment
	available   map[uint64]int
	reserved    map[uint64]int
	nextUser    uint64
	nextOrder   uint64
	nextPayment uint64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users:       map[uint64]*user.User{},
		usersEmail:  map[string]uint64{},
		products:    map[uint64]*product.Product{},
		orders:      map[uint64]*order.Order{},
		ordersByKey: map[string]uint64{},
		payments:    map[uint64]*payment.Payment{},
		available:   map[uint64]int{},
		reserved:    map[uint64]int{},
	}
}

type fakeProvider struct {
	store *fakeStore
}

func (p *fakeProvider) Users() ports.UserRepository       { return &fakeUserRepo{store: p.store} }
func (p *fakeProvider) Products() ports.ProductRepository { return &fakeProductRepo{store: p.store} }
func (p *fakeProvider) Inventory() ports.InventoryRepository {
	return &fakeInventoryRepo{store: p.store}
}
func (p *fakeProvider) Orders() ports.OrderRepository     { return &fakeOrderRepo{store: p.store} }
func (p *fakeProvider) Payments() ports.PaymentRepository { return &fakePaymentRepo{store: p.store} }
func (p *fakeProvider) Notifications() ports.NotificationRepository {
	return &fakeNotificationRepo{store: p.store}
}
func (p *fakeProvider) Audit() ports.AuditRepository { return &fakeAuditRepo{store: p.store} }

type fakeUoW struct {
	store *fakeStore
}

func (u *fakeUoW) Do(ctx context.Context, fn func(ports.RepositoryProvider) error) error {
	return fn(&fakeProvider{store: u.store})
}

type fakeUserRepo struct {
	store *fakeStore
}

func (r *fakeUserRepo) Create(ctx context.Context, u *user.User) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.usersEmail[u.Email()]; ok {
		return errs.Conflict("user.email_taken", "email is already registered")
	}
	r.store.nextUser++
	u.AssignID(r.store.nextUser)
	r.store.users[u.ID()] = u
	r.store.usersEmail[u.Email()] = u.ID()
	return nil
}

func (r *fakeUserRepo) Update(ctx context.Context, u *user.User) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.users[u.ID()] = u
	return nil
}

func (r *fakeUserRepo) FindByID(ctx context.Context, id uint64) (*user.User, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	u, ok := r.store.users[id]
	if !ok {
		return nil, errs.NotFound("user.not_found", "user not found")
	}
	return u, nil
}

func (r *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id, ok := r.store.usersEmail[email]
	if !ok {
		return nil, errs.NotFound("user.not_found", "user not found")
	}
	return r.store.users[id], nil
}

type fakeProductRepo struct {
	store *fakeStore
}

func (r *fakeProductRepo) Create(ctx context.Context, p *product.Product) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.products[p.ID()] = p
	return nil
}

func (r *fakeProductRepo) Update(ctx context.Context, p *product.Product) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.products[p.ID()] = p
	return nil
}

func (r *fakeProductRepo) FindByID(ctx context.Context, id uint64) (*product.Product, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	p, ok := r.store.products[id]
	if !ok {
		return nil, errs.NotFound("product.not_found", "product not found")
	}
	return p, nil
}

func (r *fakeProductRepo) List(ctx context.Context, page ports.Page) ([]*product.Product, int64, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	out := make([]*product.Product, 0, len(r.store.products))
	for _, p := range r.store.products {
		out = append(out, p)
	}
	return out, int64(len(out)), nil
}

type fakeInventoryRepo struct {
	store *fakeStore
}

func (r *fakeInventoryRepo) FindByProductID(ctx context.Context, productID uint64) (*inventory.Inventory, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if _, ok := r.store.available[productID]; !ok {
		return nil, errs.NotFound("inventory.not_found", "inventory not found")
	}
	return inventory.ReconstituteInventory(productID, productID, r.store.available[productID], r.store.reserved[productID], 0, 0), nil
}

func (r *fakeInventoryRepo) Save(ctx context.Context, inv *inventory.Inventory) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.available[inv.ProductID()] = inv.Available()
	r.store.reserved[inv.ProductID()] = inv.Reserved()
	return nil
}

func (r *fakeInventoryRepo) Reserve(ctx context.Context, productID uint64, qty int) (bool, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if r.store.available[productID] < qty {
		return false, nil
	}
	r.store.available[productID] -= qty
	r.store.reserved[productID] += qty
	return true, nil
}

func (r *fakeInventoryRepo) Release(ctx context.Context, productID uint64, qty int) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.reserved[productID] -= qty
	r.store.available[productID] += qty
	return nil
}

func (r *fakeInventoryRepo) Commit(ctx context.Context, productID uint64, qty int) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.reserved[productID] -= qty
	return nil
}

func (r *fakeInventoryRepo) Restock(ctx context.Context, productID uint64, qty int) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.available[productID] += qty
	return nil
}

func (r *fakeInventoryRepo) ListLowStock(ctx context.Context) ([]*inventory.Inventory, error) {
	return nil, nil
}

type fakeOrderRepo struct {
	store *fakeStore
}

func (r *fakeOrderRepo) Create(ctx context.Context, o *order.Order) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.nextOrder++
	o.AssignID(r.store.nextOrder)
	r.store.orders[o.ID()] = o
	r.store.ordersByKey[o.IdempotencyKey()] = o.ID()
	return nil
}

func (r *fakeOrderRepo) Update(ctx context.Context, o *order.Order) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.orders[o.ID()] = o
	return nil
}

func (r *fakeOrderRepo) FindByID(ctx context.Context, id uint64) (*order.Order, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	o, ok := r.store.orders[id]
	if !ok {
		return nil, errs.NotFound("order.not_found", "order not found")
	}
	return o, nil
}

func (r *fakeOrderRepo) FindByIdempotencyKey(ctx context.Context, key string) (*order.Order, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	id, ok := r.store.ordersByKey[key]
	if !ok {
		return nil, errs.NotFound("order.not_found", "order not found")
	}
	return r.store.orders[id], nil
}

func (r *fakeOrderRepo) ListByUser(ctx context.Context, userID uint64, page ports.Page) ([]*order.Order, int64, error) {
	return nil, 0, nil
}

func (r *fakeOrderRepo) ListAll(ctx context.Context, page ports.Page) ([]*order.Order, int64, error) {
	return nil, 0, nil
}

type fakePaymentRepo struct {
	store *fakeStore
}

func (r *fakePaymentRepo) Create(ctx context.Context, p *payment.Payment) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.nextPayment++
	p.AssignID(r.store.nextPayment)
	r.store.payments[p.ID()] = p
	return nil
}

func (r *fakePaymentRepo) Update(ctx context.Context, p *payment.Payment) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.payments[p.ID()] = p
	return nil
}

func (r *fakePaymentRepo) FindByID(ctx context.Context, id uint64) (*payment.Payment, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	p, ok := r.store.payments[id]
	if !ok {
		return nil, errs.NotFound("payment.not_found", "payment not found")
	}
	return p, nil
}

func (r *fakePaymentRepo) FindByOrderID(ctx context.Context, orderID uint64) (*payment.Payment, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	for _, p := range r.store.payments {
		if p.OrderID() == orderID {
			return p, nil
		}
	}
	return nil, errs.NotFound("payment.not_found", "payment not found")
}

func (r *fakePaymentRepo) FindByIdempotencyKey(ctx context.Context, key string) (*payment.Payment, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	for _, p := range r.store.payments {
		if p.IdempotencyKey() == key {
			return p, nil
		}
	}
	return nil, errs.NotFound("payment.not_found", "payment not found")
}

type fakeNotificationRepo struct {
	store *fakeStore
}

func (r *fakeNotificationRepo) Create(ctx context.Context, n *notification.Notification) error {
	return nil
}

func (r *fakeNotificationRepo) Update(ctx context.Context, n *notification.Notification) error {
	return nil
}

func (r *fakeNotificationRepo) ListByStatus(ctx context.Context, status notification.Status, limit int) ([]*notification.Notification, error) {
	return nil, nil
}

type fakeAuditRepo struct {
	store *fakeStore
}

func (r *fakeAuditRepo) Append(ctx context.Context, e *audit.Entry) error {
	return nil
}

type fakePaymentGateway struct {
	mu            sync.Mutex
	declineAlways bool
	failFirst     int
	calls         int
	results       map[string]ports.ChargeResult
}

func newFakePaymentGateway(decline bool) *fakePaymentGateway {
	return &fakePaymentGateway{declineAlways: decline, results: map[string]ports.ChargeResult{}}
}

func newFlakyGateway(failFirst int) *fakePaymentGateway {
	return &fakePaymentGateway{failFirst: failFirst, results: map[string]ports.ChargeResult{}}
}

func (g *fakePaymentGateway) Charge(ctx context.Context, req ports.ChargeRequest) (ports.ChargeResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if existing, ok := g.results[req.IdempotencyKey]; ok {
		return existing, nil
	}
	g.calls++
	if g.declineAlways || g.calls <= g.failFirst {
		return ports.ChargeResult{}, errs.PaymentFailed("payment.declined", "payment was declined by the gateway")
	}
	res := ports.ChargeResult{Provider: "mock", ProviderRef: "ref-" + req.IdempotencyKey}
	g.results[req.IdempotencyKey] = res
	return res, nil
}

func (g *fakePaymentGateway) chargeCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls
}

type fakeEventBus struct {
	mu        sync.Mutex
	published []shared.Event
}

func (b *fakeEventBus) Publish(ctx context.Context, events ...shared.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, events...)
	return nil
}

func (b *fakeEventBus) Subscribe(handler ports.EventHandler) {}

func (b *fakeEventBus) names() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, len(b.published))
	for _, e := range b.published {
		out = append(out, e.EventName())
	}
	return out
}

func newStartedPool() *concurrency.Pool {
	p := concurrency.NewPool(2, 8)
	p.Start()
	return p
}

func seedProduct(t *testing.T, store *fakeStore, id uint64, price int64, available int) {
	t.Helper()
	p, err := product.NewProduct("SKU-"+itoa(id), "Product "+itoa(id), "", shared.MustMoney(price, shared.DefaultCurrency))
	require.NoError(t, err)
	p.AssignID(id)
	store.products[id] = p
	store.available[id] = available
	store.reserved[id] = 0
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func TestPlaceOrderHappyPath(t *testing.T) {
	store := newFakeStore()
	seedProduct(t, store, 1, 1000, 10)

	pool := newStartedPool()
	defer pool.Stop()

	gateway := newFakePaymentGateway(false)
	bus := &fakeEventBus{}
	provider := &fakeProvider{store: store}

	h := NewPlaceOrder(&fakeUoW{store: store}, provider, gateway, bus, pool, 3, 0)

	ord, err := h.Handle(context.Background(), PlaceOrderInput{
		UserID:         42,
		IdempotencyKey: "key-happy",
		Items:          []PlaceOrderItem{{ProductID: 1, Quantity: 3}},
	})

	require.NoError(t, err)
	require.NotNil(t, ord)
	assert.Equal(t, order.StatusFulfilled, ord.Status())
	assert.Equal(t, 7, store.available[1])
	assert.Equal(t, 0, store.reserved[1])

	pay, err := provider.Payments().FindByOrderID(context.Background(), ord.ID())
	require.NoError(t, err)
	assert.Equal(t, payment.StatusSucceeded, pay.Status())

	names := bus.names()
	assert.Contains(t, names, "order.paid")
	assert.Contains(t, names, "order.fulfilled")
	assert.Contains(t, names, "payment.succeeded")
}

func TestPlaceOrderOutOfStock(t *testing.T) {
	store := newFakeStore()
	seedProduct(t, store, 1, 1000, 2)

	pool := newStartedPool()
	defer pool.Stop()

	gateway := newFakePaymentGateway(false)
	bus := &fakeEventBus{}
	provider := &fakeProvider{store: store}

	h := NewPlaceOrder(&fakeUoW{store: store}, provider, gateway, bus, pool, 3, 0)

	ord, err := h.Handle(context.Background(), PlaceOrderInput{
		UserID:         42,
		IdempotencyKey: "key-oos",
		Items:          []PlaceOrderItem{{ProductID: 1, Quantity: 5}},
	})

	require.Error(t, err)
	assert.Nil(t, ord)
	assert.Equal(t, errs.KindOutOfStock, errs.KindOf(err))

	assert.Equal(t, 2, store.available[1])
	assert.Equal(t, 0, store.reserved[1])

	for _, o := range store.orders {
		assert.NotEqual(t, order.StatusReserved, o.Status())
	}
	assert.Equal(t, 0, gateway.chargeCount())
}

func TestPlaceOrderPaymentDeclined(t *testing.T) {
	store := newFakeStore()
	seedProduct(t, store, 1, 1000, 10)

	pool := newStartedPool()
	defer pool.Stop()

	gateway := newFakePaymentGateway(true)
	bus := &fakeEventBus{}
	provider := &fakeProvider{store: store}

	h := NewPlaceOrder(&fakeUoW{store: store}, provider, gateway, bus, pool, 3, 0)

	ord, err := h.Handle(context.Background(), PlaceOrderInput{
		UserID:         42,
		IdempotencyKey: "key-declined",
		Items:          []PlaceOrderItem{{ProductID: 1, Quantity: 4}},
	})

	require.Error(t, err)
	assert.Nil(t, ord)
	assert.Equal(t, errs.KindPayment, errs.KindOf(err))

	require.Len(t, store.orders, 1)
	var stored *order.Order
	for _, o := range store.orders {
		stored = o
	}
	assert.Equal(t, order.StatusFailed, stored.Status())

	assert.Equal(t, 10, store.available[1])
	assert.Equal(t, 0, store.reserved[1])

	pay, err := provider.Payments().FindByOrderID(context.Background(), stored.ID())
	require.NoError(t, err)
	assert.Equal(t, payment.StatusFailed, pay.Status())
	assert.Equal(t, 3, gateway.chargeCount())

	names := bus.names()
	assert.Contains(t, names, "payment.failed")
	assert.Contains(t, names, "order.failed")
	assert.NotContains(t, names, "order.fulfilled")
}

func TestPlaceOrderIdempotency(t *testing.T) {
	store := newFakeStore()
	seedProduct(t, store, 1, 1000, 10)

	pool := newStartedPool()
	defer pool.Stop()

	gateway := newFakePaymentGateway(false)
	bus := &fakeEventBus{}
	provider := &fakeProvider{store: store}

	h := NewPlaceOrder(&fakeUoW{store: store}, provider, gateway, bus, pool, 3, 0)

	in := PlaceOrderInput{
		UserID:         42,
		IdempotencyKey: "key-idem",
		Items:          []PlaceOrderItem{{ProductID: 1, Quantity: 3}},
	}

	first, err := h.Handle(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := h.Handle(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, second)

	assert.Equal(t, first.ID(), second.ID())
	assert.Equal(t, 1, gateway.chargeCount())
	assert.Len(t, store.orders, 1)
	assert.Equal(t, 7, store.available[1])
}

func TestPlaceOrderRetriesThenSucceeds(t *testing.T) {
	store := newFakeStore()
	seedProduct(t, store, 1, 1000, 10)

	pool := newStartedPool()
	defer pool.Stop()

	gateway := newFlakyGateway(2)
	bus := &fakeEventBus{}
	provider := &fakeProvider{store: store}

	h := NewPlaceOrder(&fakeUoW{store: store}, provider, gateway, bus, pool, 3, 0)

	ord, err := h.Handle(context.Background(), PlaceOrderInput{
		UserID:         42,
		IdempotencyKey: "key-retry",
		Items:          []PlaceOrderItem{{ProductID: 1, Quantity: 2}},
	})

	require.NoError(t, err)
	require.NotNil(t, ord)
	assert.Equal(t, order.StatusFulfilled, ord.Status())
	assert.Equal(t, 3, gateway.chargeCount())
	assert.Equal(t, 8, store.available[1])
}
