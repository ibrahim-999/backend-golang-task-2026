package order

import (
	"testing"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustItem(t *testing.T, productID uint64, sku, name string, amount int64, currency string, quantity int) Item {
	t.Helper()
	item, err := NewItem(productID, sku, name, shared.MustMoney(amount, currency), quantity)
	require.NoError(t, err)
	return item
}

func TestNewItem(t *testing.T) {
	t.Run("valid item computes subtotal", func(t *testing.T) {
		item, err := NewItem(1, "SKU-1", "Widget", shared.MustMoney(250, "USD"), 3)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), item.ProductID())
		assert.Equal(t, "SKU-1", item.SKU())
		assert.Equal(t, "Widget", item.Name())
		assert.Equal(t, int64(250), item.UnitPrice().Amount())
		assert.Equal(t, 3, item.Quantity())
		assert.True(t, item.Subtotal().Equal(shared.MustMoney(750, "USD")))
	})

	t.Run("rejects non-positive quantity", func(t *testing.T) {
		for _, qty := range []int{0, -1, -5} {
			_, err := NewItem(1, "SKU-1", "Widget", shared.MustMoney(250, "USD"), qty)
			require.Error(t, err)
			assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		}
	})
}

func TestNewOrder(t *testing.T) {
	t.Run("builds pending order and records OrderPlaced", func(t *testing.T) {
		items := []Item{
			mustItem(t, 1, "SKU-1", "Widget", 250, "USD", 2),
			mustItem(t, 2, "SKU-2", "Gadget", 100, "USD", 3),
		}
		o, err := NewOrder(42, items, "idem-key-1")
		require.NoError(t, err)

		assert.Equal(t, uint64(42), o.UserID())
		assert.Equal(t, StatusPending, o.Status())
		assert.Equal(t, "idem-key-1", o.IdempotencyKey())
		assert.Len(t, o.Items(), 2)
		assert.True(t, o.Total().Equal(shared.MustMoney(800, "USD")))
		assert.False(t, o.CreatedAt().IsZero())

		events := o.PullEvents()
		require.Len(t, events, 1)
		placed, ok := events[0].(OrderPlaced)
		require.True(t, ok)
		assert.Equal(t, "order.placed", placed.EventName())
		assert.Equal(t, uint64(42), placed.UserID)
		assert.Equal(t, int64(800), placed.TotalAmount)
		assert.Equal(t, "USD", placed.Currency)
		assert.False(t, o.HasPendingEvents())
	})

	t.Run("single item total", func(t *testing.T) {
		o, err := NewOrder(1, []Item{mustItem(t, 1, "SKU-1", "Widget", 500, "USD", 1)}, "k")
		require.NoError(t, err)
		assert.True(t, o.Total().Equal(shared.MustMoney(500, "USD")))
	})

	t.Run("rejects empty items", func(t *testing.T) {
		_, err := NewOrder(1, nil, "k")
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))

		_, err = NewOrder(1, []Item{}, "k")
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
	})

	t.Run("rejects empty idempotency key", func(t *testing.T) {
		_, err := NewOrder(1, []Item{mustItem(t, 1, "SKU-1", "Widget", 500, "USD", 1)}, "")
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
	})

	t.Run("rejects currency mismatch", func(t *testing.T) {
		items := []Item{
			mustItem(t, 1, "SKU-1", "Widget", 250, "USD", 1),
			mustItem(t, 2, "SKU-2", "Gadget", 100, "EUR", 1),
		}
		_, err := NewOrder(1, items, "k")
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		e, ok := errs.As(err)
		require.True(t, ok)
		assert.Equal(t, "order.currency_mismatch", e.Code)
	})
}

func TestReconstituteOrder(t *testing.T) {
	created := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	items := []Item{mustItem(t, 1, "SKU-1", "Widget", 500, "USD", 1)}
	o := ReconstituteOrder(7, 9, StatusPaid, items, shared.MustMoney(500, "USD"), "key", "reason", created)

	assert.Equal(t, uint64(7), o.ID())
	assert.Equal(t, uint64(9), o.UserID())
	assert.Equal(t, StatusPaid, o.Status())
	assert.Equal(t, "key", o.IdempotencyKey())
	assert.Equal(t, "reason", o.FailureReason())
	assert.Equal(t, created, o.CreatedAt())
	assert.False(t, o.HasPendingEvents())
}

func newPendingOrder(t *testing.T) *Order {
	t.Helper()
	o, err := NewOrder(1, []Item{mustItem(t, 1, "SKU-1", "Widget", 500, "USD", 1)}, "k")
	require.NoError(t, err)
	o.PullEvents()
	return o
}

func TestValidTransitions(t *testing.T) {
	t.Run("pending to reserved", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.MarkReserved())
		assert.Equal(t, StatusReserved, o.Status())
		events := o.PullEvents()
		require.Len(t, events, 1)
		ev, ok := events[0].(OrderReserved)
		require.True(t, ok)
		assert.Equal(t, "order.reserved", ev.EventName())
	})

	t.Run("reserved to paid", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.MarkReserved())
		o.PullEvents()
		require.NoError(t, o.MarkPaid())
		assert.Equal(t, StatusPaid, o.Status())
		events := o.PullEvents()
		require.Len(t, events, 1)
		_, ok := events[0].(OrderPaid)
		assert.True(t, ok)
	})

	t.Run("paid to fulfilled", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.MarkReserved())
		require.NoError(t, o.MarkPaid())
		o.PullEvents()
		require.NoError(t, o.MarkFulfilled())
		assert.Equal(t, StatusFulfilled, o.Status())
		assert.True(t, o.IsTerminal())
		events := o.PullEvents()
		require.Len(t, events, 1)
		_, ok := events[0].(OrderFulfilled)
		assert.True(t, ok)
	})

	t.Run("full happy path", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.MarkReserved())
		require.NoError(t, o.MarkPaid())
		require.NoError(t, o.MarkFulfilled())
		assert.Equal(t, StatusFulfilled, o.Status())
	})
}

func TestCancel(t *testing.T) {
	t.Run("from pending", func(t *testing.T) {
		o := newPendingOrder(t)
		assert.True(t, o.CanCancel())
		require.NoError(t, o.Cancel("changed mind"))
		assert.Equal(t, StatusCancelled, o.Status())
		assert.True(t, o.IsTerminal())
		events := o.PullEvents()
		require.Len(t, events, 1)
		ev, ok := events[0].(OrderCancelled)
		require.True(t, ok)
		assert.Equal(t, "changed mind", ev.Reason)
		assert.Equal(t, "order.cancelled", ev.EventName())
	})

	t.Run("from reserved", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.MarkReserved())
		o.PullEvents()
		require.NoError(t, o.Cancel("oops"))
		assert.Equal(t, StatusCancelled, o.Status())
	})

	t.Run("from paid", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.MarkReserved())
		require.NoError(t, o.MarkPaid())
		o.PullEvents()
		require.NoError(t, o.Cancel("refund"))
		assert.Equal(t, StatusCancelled, o.Status())
	})

	t.Run("cannot cancel terminal", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.MarkReserved())
		require.NoError(t, o.MarkPaid())
		require.NoError(t, o.MarkFulfilled())
		o.PullEvents()
		assert.False(t, o.CanCancel())
		err := o.Cancel("late")
		require.Error(t, err)
		assert.Equal(t, errs.KindConflict, errs.KindOf(err))
		assert.False(t, o.HasPendingEvents())
	})
}

func TestFail(t *testing.T) {
	t.Run("from pending", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.Fail("payment declined"))
		assert.Equal(t, StatusFailed, o.Status())
		assert.Equal(t, "payment declined", o.FailureReason())
		assert.True(t, o.IsTerminal())
		events := o.PullEvents()
		require.Len(t, events, 1)
		ev, ok := events[0].(OrderFailed)
		require.True(t, ok)
		assert.Equal(t, "payment declined", ev.Reason)
		assert.Equal(t, "order.failed", ev.EventName())
	})

	t.Run("from reserved and paid", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.MarkReserved())
		require.NoError(t, o.Fail("stock gone"))
		assert.Equal(t, StatusFailed, o.Status())

		o2 := newPendingOrder(t)
		require.NoError(t, o2.MarkReserved())
		require.NoError(t, o2.MarkPaid())
		require.NoError(t, o2.Fail("fulfillment error"))
		assert.Equal(t, StatusFailed, o2.Status())
	})

	t.Run("cannot fail terminal", func(t *testing.T) {
		o := newPendingOrder(t)
		require.NoError(t, o.Cancel("x"))
		o.PullEvents()
		err := o.Fail("too late")
		require.Error(t, err)
		assert.Equal(t, errs.KindConflict, errs.KindOf(err))
		assert.Equal(t, StatusCancelled, o.Status())
		assert.False(t, o.HasPendingEvents())
	})
}

func TestInvalidTransitions(t *testing.T) {
	cases := []struct {
		name   string
		status Status
		act    func(*Order) error
	}{
		{"mark paid on pending", StatusPending, (*Order).MarkPaid},
		{"mark fulfilled on pending", StatusPending, (*Order).MarkFulfilled},
		{"mark reserved on reserved", StatusReserved, (*Order).MarkReserved},
		{"mark fulfilled on reserved", StatusReserved, (*Order).MarkFulfilled},
		{"mark reserved on paid", StatusPaid, (*Order).MarkReserved},
		{"mark paid on paid", StatusPaid, (*Order).MarkPaid},
		{"mark reserved on fulfilled", StatusFulfilled, (*Order).MarkReserved},
		{"mark paid on fulfilled", StatusFulfilled, (*Order).MarkPaid},
		{"mark fulfilled on fulfilled", StatusFulfilled, (*Order).MarkFulfilled},
		{"mark reserved on cancelled", StatusCancelled, (*Order).MarkReserved},
		{"mark reserved on failed", StatusFailed, (*Order).MarkReserved},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := ReconstituteOrder(1, 1, tc.status, nil, shared.ZeroMoney("USD"), "k", "", time.Now())
			err := tc.act(o)
			require.Error(t, err)
			assert.Equal(t, errs.KindConflict, errs.KindOf(err))
			e, ok := errs.As(err)
			require.True(t, ok)
			assert.Equal(t, "order.invalid_transition", e.Code)
			assert.Equal(t, tc.status, o.Status())
			assert.False(t, o.HasPendingEvents())
		})
	}
}

func TestIsTerminalAndCanCancel(t *testing.T) {
	cases := []struct {
		status     Status
		terminal   bool
		cancelable bool
	}{
		{StatusPending, false, true},
		{StatusReserved, false, true},
		{StatusPaid, false, true},
		{StatusFulfilled, true, false},
		{StatusCancelled, true, false},
		{StatusFailed, true, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			o := ReconstituteOrder(1, 1, tc.status, nil, shared.ZeroMoney("USD"), "k", "", time.Now())
			assert.Equal(t, tc.terminal, o.IsTerminal())
			assert.Equal(t, tc.cancelable, o.CanCancel())
		})
	}
}

func TestEventAggregateID(t *testing.T) {
	o := ReconstituteOrder(99, 1, StatusPending, nil, shared.ZeroMoney("USD"), "k", "", time.Now())
	require.NoError(t, o.MarkReserved())
	events := o.PullEvents()
	require.Len(t, events, 1)
	assert.Equal(t, uint64(99), events[0].AggregateID())
}
