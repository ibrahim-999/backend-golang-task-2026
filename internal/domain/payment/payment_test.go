package payment

import (
	"testing"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validAmount() shared.Money {
	return shared.MustMoney(5000, "USD")
}

func TestNewPayment(t *testing.T) {
	tests := []struct {
		name           string
		orderID        uint64
		idempotencyKey string
		amount         shared.Money
		wantErr        bool
		wantKind       errs.Kind
	}{
		{
			name:           "valid payment",
			orderID:        42,
			idempotencyKey: "key-1",
			amount:         validAmount(),
			wantErr:        false,
		},
		{
			name:           "zero order id",
			orderID:        0,
			idempotencyKey: "key-1",
			amount:         validAmount(),
			wantErr:        true,
			wantKind:       errs.KindValidation,
		},
		{
			name:           "empty idempotency key",
			orderID:        42,
			idempotencyKey: "",
			amount:         validAmount(),
			wantErr:        true,
			wantKind:       errs.KindValidation,
		},
		{
			name:           "zero amount",
			orderID:        42,
			idempotencyKey: "key-1",
			amount:         shared.ZeroMoney("USD"),
			wantErr:        true,
			wantKind:       errs.KindValidation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewPayment(tc.orderID, tc.idempotencyKey, tc.amount)
			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, p)
				assert.Equal(t, tc.wantKind, errs.KindOf(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, p)
			assert.Equal(t, tc.orderID, p.OrderID())
			assert.Equal(t, tc.idempotencyKey, p.IdempotencyKey())
			assert.True(t, p.Amount().Equal(tc.amount))
			assert.Equal(t, StatusPending, p.Status())
			assert.Equal(t, 0, p.Attempts())
			assert.Empty(t, p.Provider())
			assert.Empty(t, p.ProviderRef())
			assert.Empty(t, p.FailureReason())
			assert.False(t, p.CreatedAt().IsZero())

			events := p.PullEvents()
			require.Len(t, events, 1)
			initiated, ok := events[0].(PaymentInitiated)
			require.True(t, ok)
			assert.Equal(t, "payment.initiated", initiated.EventName())
			assert.Equal(t, p.OrderID(), initiated.OrderID)
			assert.Equal(t, p.ID(), initiated.AggregateID())
			assert.False(t, p.HasPendingEvents())
		})
	}
}

func TestReconstitutePayment(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	p := ReconstitutePayment(
		7,
		42,
		"key-1",
		validAmount(),
		StatusSucceeded,
		"stripe",
		"ref-123",
		2,
		"",
		created,
	)

	assert.Equal(t, uint64(7), p.ID())
	assert.Equal(t, uint64(42), p.OrderID())
	assert.Equal(t, "key-1", p.IdempotencyKey())
	assert.True(t, p.Amount().Equal(validAmount()))
	assert.Equal(t, StatusSucceeded, p.Status())
	assert.Equal(t, "stripe", p.Provider())
	assert.Equal(t, "ref-123", p.ProviderRef())
	assert.Equal(t, 2, p.Attempts())
	assert.Empty(t, p.FailureReason())
	assert.Equal(t, created, p.CreatedAt())
	assert.False(t, p.HasPendingEvents())
}

func TestRecordAttempt(t *testing.T) {
	t.Run("increments and sets provider", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		p.PullEvents()

		require.NoError(t, p.RecordAttempt("stripe"))
		assert.Equal(t, 1, p.Attempts())
		assert.Equal(t, "stripe", p.Provider())

		require.NoError(t, p.RecordAttempt("adyen"))
		assert.Equal(t, 2, p.Attempts())
		assert.Equal(t, "adyen", p.Provider())
		assert.False(t, p.HasPendingEvents())
	})

	t.Run("empty provider rejected", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)

		err = p.RecordAttempt("")
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.Equal(t, 0, p.Attempts())
	})
}

func TestMarkSucceeded(t *testing.T) {
	t.Run("pending to succeeded", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		p.PullEvents()

		require.NoError(t, p.MarkSucceeded("ref-123"))
		assert.Equal(t, StatusSucceeded, p.Status())
		assert.Equal(t, "ref-123", p.ProviderRef())
		assert.Empty(t, p.FailureReason())

		events := p.PullEvents()
		require.Len(t, events, 1)
		succeeded, ok := events[0].(PaymentSucceeded)
		require.True(t, ok)
		assert.Equal(t, "payment.succeeded", succeeded.EventName())
		assert.Equal(t, p.OrderID(), succeeded.OrderID)
		assert.Equal(t, p.ID(), succeeded.AggregateID())
	})

	t.Run("empty provider ref rejected", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		p.PullEvents()

		err = p.MarkSucceeded("")
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.Equal(t, StatusPending, p.Status())
		assert.False(t, p.HasPendingEvents())
	})

	t.Run("not pending rejected", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		require.NoError(t, p.MarkFailed("declined"))
		p.PullEvents()

		err = p.MarkSucceeded("ref-123")
		require.Error(t, err)
		assert.Equal(t, errs.KindConflict, errs.KindOf(err))
		assert.Equal(t, StatusFailed, p.Status())
		assert.False(t, p.HasPendingEvents())
	})
}

func TestMarkFailed(t *testing.T) {
	t.Run("pending to failed", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		p.PullEvents()

		require.NoError(t, p.MarkFailed("card declined"))
		assert.Equal(t, StatusFailed, p.Status())
		assert.Equal(t, "card declined", p.FailureReason())

		events := p.PullEvents()
		require.Len(t, events, 1)
		failed, ok := events[0].(PaymentFailed)
		require.True(t, ok)
		assert.Equal(t, "payment.failed", failed.EventName())
		assert.Equal(t, "card declined", failed.Reason)
		assert.Equal(t, p.OrderID(), failed.OrderID)
		assert.Equal(t, p.ID(), failed.AggregateID())
	})

	t.Run("empty reason rejected", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		p.PullEvents()

		err = p.MarkFailed("")
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.Equal(t, StatusPending, p.Status())
		assert.False(t, p.HasPendingEvents())
	})

	t.Run("not pending rejected", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		require.NoError(t, p.MarkSucceeded("ref-123"))
		p.PullEvents()

		err = p.MarkFailed("late")
		require.Error(t, err)
		assert.Equal(t, errs.KindConflict, errs.KindOf(err))
		assert.Equal(t, StatusSucceeded, p.Status())
		assert.False(t, p.HasPendingEvents())
	})
}

func TestRefund(t *testing.T) {
	t.Run("succeeded to refunded", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		require.NoError(t, p.MarkSucceeded("ref-123"))
		p.PullEvents()

		require.NoError(t, p.Refund())
		assert.Equal(t, StatusRefunded, p.Status())

		events := p.PullEvents()
		require.Len(t, events, 1)
		refunded, ok := events[0].(PaymentRefunded)
		require.True(t, ok)
		assert.Equal(t, "payment.refunded", refunded.EventName())
		assert.Equal(t, p.OrderID(), refunded.OrderID)
		assert.Equal(t, p.ID(), refunded.AggregateID())
	})

	t.Run("pending cannot refund", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		p.PullEvents()

		err = p.Refund()
		require.Error(t, err)
		assert.Equal(t, errs.KindConflict, errs.KindOf(err))
		assert.Equal(t, StatusPending, p.Status())
		assert.False(t, p.HasPendingEvents())
	})

	t.Run("failed cannot refund", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		require.NoError(t, p.MarkFailed("declined"))
		p.PullEvents()

		err = p.Refund()
		require.Error(t, err)
		assert.Equal(t, errs.KindConflict, errs.KindOf(err))
		assert.Equal(t, StatusFailed, p.Status())
		assert.False(t, p.HasPendingEvents())
	})

	t.Run("already refunded cannot refund again", func(t *testing.T) {
		p, err := NewPayment(42, "key-1", validAmount())
		require.NoError(t, err)
		require.NoError(t, p.MarkSucceeded("ref-123"))
		require.NoError(t, p.Refund())
		p.PullEvents()

		err = p.Refund()
		require.Error(t, err)
		assert.Equal(t, errs.KindConflict, errs.KindOf(err))
		assert.Equal(t, StatusRefunded, p.Status())
		assert.False(t, p.HasPendingEvents())
	})
}

func TestFullLifecycle(t *testing.T) {
	p, err := NewPayment(99, "idem-xyz", validAmount())
	require.NoError(t, err)

	require.NoError(t, p.RecordAttempt("stripe"))
	require.NoError(t, p.MarkSucceeded("ch_123"))
	require.NoError(t, p.Refund())

	assert.Equal(t, StatusRefunded, p.Status())
	assert.Equal(t, 1, p.Attempts())
	assert.Equal(t, "stripe", p.Provider())
	assert.Equal(t, "ch_123", p.ProviderRef())

	events := p.PullEvents()
	require.Len(t, events, 3)
	assert.Equal(t, "payment.initiated", events[0].EventName())
	assert.Equal(t, "payment.succeeded", events[1].EventName())
	assert.Equal(t, "payment.refunded", events[2].EventName())
}
