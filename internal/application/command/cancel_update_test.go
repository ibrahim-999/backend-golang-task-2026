package command

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/payment"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

func putOrder(store *fakeStore, id, productID uint64, status order.Status, qty int) *order.Order {
	item, _ := order.NewItem(productID, "SKU", "Name", shared.MustMoney(1000, "USD"), qty)
	ord := order.ReconstituteOrder(id, 42, status, []order.Item{item}, shared.MustMoney(int64(1000*qty), "USD"), "key-"+itoa(id), "", time.Now())
	store.orders[id] = ord
	store.ordersByKey[ord.IdempotencyKey()] = id
	return ord
}

func TestCancelOrderReservedReleasesStock(t *testing.T) {
	store := newFakeStore()
	store.available[1] = 8
	store.reserved[1] = 2
	putOrder(store, 1, 1, order.StatusReserved, 2)

	bus := &fakeEventBus{}
	h := NewCancelOrder(&fakeUoW{store: store}, &fakeProvider{store: store}, bus)

	ord, err := h.Handle(context.Background(), CancelOrderInput{OrderID: 1, Reason: "changed mind"})
	require.NoError(t, err)
	assert.Equal(t, order.StatusCancelled, ord.Status())
	assert.Equal(t, 10, store.available[1])
	assert.Equal(t, 0, store.reserved[1])
	assert.Contains(t, bus.names(), "order.cancelled")
}

func TestCancelOrderPaidRefundsAndReleases(t *testing.T) {
	store := newFakeStore()
	store.available[1] = 8
	store.reserved[1] = 2
	putOrder(store, 1, 1, order.StatusPaid, 2)
	store.payments[1] = payment.ReconstitutePayment(1, 1, "key-1", shared.MustMoney(2000, "USD"), payment.StatusSucceeded, "mock", "ref-1", 1, "", time.Now())

	h := NewCancelOrder(&fakeUoW{store: store}, &fakeProvider{store: store}, &fakeEventBus{})

	ord, err := h.Handle(context.Background(), CancelOrderInput{OrderID: 1, Reason: "refund please"})
	require.NoError(t, err)
	assert.Equal(t, order.StatusCancelled, ord.Status())
	assert.Equal(t, 10, store.available[1])
	assert.Equal(t, payment.StatusRefunded, store.payments[1].Status())
}

func TestCancelOrderFulfilledRejected(t *testing.T) {
	store := newFakeStore()
	putOrder(store, 1, 1, order.StatusFulfilled, 1)

	h := NewCancelOrder(&fakeUoW{store: store}, &fakeProvider{store: store}, &fakeEventBus{})

	_, err := h.Handle(context.Background(), CancelOrderInput{OrderID: 1, Reason: "too late"})
	require.Error(t, err)
}

func TestCancelOrderNotFound(t *testing.T) {
	store := newFakeStore()
	h := NewCancelOrder(&fakeUoW{store: store}, &fakeProvider{store: store}, &fakeEventBus{})

	_, err := h.Handle(context.Background(), CancelOrderInput{OrderID: 999, Reason: "x"})
	assert.Equal(t, errs.KindNotFound, errs.KindOf(err))
}

func TestUpdateOrderStatusTransitions(t *testing.T) {
	store := newFakeStore()
	h := NewUpdateOrderStatus(&fakeUoW{store: store})
	ctx := context.Background()

	putOrder(store, 1, 1, order.StatusPending, 1)
	require.NoError(t, h.Handle(ctx, 1, order.StatusReserved))
	assert.Equal(t, order.StatusReserved, store.orders[1].Status())

	putOrder(store, 2, 1, order.StatusReserved, 1)
	require.NoError(t, h.Handle(ctx, 2, order.StatusPaid))

	putOrder(store, 3, 1, order.StatusPaid, 1)
	require.NoError(t, h.Handle(ctx, 3, order.StatusFulfilled))

	putOrder(store, 4, 1, order.StatusReserved, 1)
	require.NoError(t, h.Handle(ctx, 4, order.StatusCancelled))

	putOrder(store, 5, 1, order.StatusPending, 1)
	require.NoError(t, h.Handle(ctx, 5, order.StatusFailed))

	putOrder(store, 6, 1, order.StatusPending, 1)
	require.Error(t, h.Handle(ctx, 6, order.Status("bogus")))

	require.Error(t, h.Handle(ctx, 999, order.StatusReserved))
}
