package notification

import (
	"testing"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func userIDPtr(v uint64) *uint64 {
	return &v
}

func eventNames(events []shared.Event) []string {
	names := make([]string, 0, len(events))
	for _, e := range events {
		names = append(names, e.EventName())
	}
	return names
}

func TestNewNotification(t *testing.T) {
	uid := userIDPtr(42)

	tests := []struct {
		name    string
		userID  *uint64
		ntype   Type
		channel Channel
		subject string
		payload string
		wantErr bool
		errKind errs.Kind
	}{
		{
			name:    "valid with user",
			userID:  uid,
			ntype:   TypeOrderConfirmed,
			channel: ChannelEmail,
			subject: "Your order is confirmed",
			payload: "{}",
		},
		{
			name:    "valid without user",
			userID:  nil,
			ntype:   TypeLowStock,
			channel: ChannelPush,
			subject: "Stock running low",
			payload: "",
		},
		{
			name:    "invalid type",
			userID:  uid,
			ntype:   Type("bogus"),
			channel: ChannelEmail,
			subject: "x",
			wantErr: true,
			errKind: errs.KindValidation,
		},
		{
			name:    "invalid channel",
			userID:  uid,
			ntype:   TypeOrderFailed,
			channel: Channel("carrier-pigeon"),
			subject: "x",
			wantErr: true,
			errKind: errs.KindValidation,
		},
		{
			name:    "empty subject",
			userID:  uid,
			ntype:   TypePaymentFailed,
			channel: ChannelSMS,
			subject: "",
			wantErr: true,
			errKind: errs.KindValidation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, err := NewNotification(tc.userID, tc.ntype, tc.channel, tc.subject, tc.payload)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, n)
				assert.Equal(t, tc.errKind, errs.KindOf(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, n)
			assert.Equal(t, tc.ntype, n.Type())
			assert.Equal(t, tc.channel, n.Channel())
			assert.Equal(t, StatusQueued, n.Status())
			assert.Equal(t, tc.subject, n.Subject())
			assert.Equal(t, tc.payload, n.Payload())
			assert.Equal(t, tc.userID, n.UserID())
			assert.Equal(t, 0, n.Attempts())
			assert.Empty(t, n.FailureReason())

			events := n.PullEvents()
			require.Len(t, events, 1)
			queued, ok := events[0].(NotificationQueued)
			require.True(t, ok)
			assert.Equal(t, "notification.queued", queued.EventName())
			assert.Equal(t, n.ID(), queued.AggregateID())
			assert.False(t, n.HasPendingEvents())
		})
	}
}

func newQueued(t *testing.T) *Notification {
	t.Helper()
	n, err := NewNotification(userIDPtr(7), TypeOrderConfirmed, ChannelEmail, "subject", "payload")
	require.NoError(t, err)
	n.PullEvents()
	return n
}

func TestIncrementAttempt(t *testing.T) {
	n := newQueued(t)
	assert.Equal(t, 0, n.Attempts())

	n.IncrementAttempt()
	n.IncrementAttempt()
	n.IncrementAttempt()

	assert.Equal(t, 3, n.Attempts())
	assert.False(t, n.HasPendingEvents())
}

func TestMarkSent(t *testing.T) {
	t.Run("from queued", func(t *testing.T) {
		n := newQueued(t)

		err := n.MarkSent()

		require.NoError(t, err)
		assert.Equal(t, StatusSent, n.Status())

		events := n.PullEvents()
		require.Len(t, events, 1)
		sent, ok := events[0].(NotificationSent)
		require.True(t, ok)
		assert.Equal(t, "notification.sent", sent.EventName())
		assert.Equal(t, n.ID(), sent.AggregateID())
	})

	t.Run("from failed clears reason", func(t *testing.T) {
		n := ReconstituteNotification(11, userIDPtr(7), TypeOrderFailed, ChannelSMS, StatusFailed, "subj", "pl", 2, "smtp down", time.Now())

		err := n.MarkSent()

		require.NoError(t, err)
		assert.Equal(t, StatusSent, n.Status())
		assert.Empty(t, n.FailureReason())
		assert.Equal(t, []string{"notification.sent"}, eventNames(n.PullEvents()))
	})

	t.Run("already sent is rejected", func(t *testing.T) {
		n := newQueued(t)
		require.NoError(t, n.MarkSent())
		n.PullEvents()

		err := n.MarkSent()

		require.Error(t, err)
		assert.Equal(t, errs.KindConflict, errs.KindOf(err))
		assert.Equal(t, StatusSent, n.Status())
		assert.False(t, n.HasPendingEvents())
	})
}

func TestMarkFailed(t *testing.T) {
	t.Run("from queued", func(t *testing.T) {
		n := newQueued(t)

		err := n.MarkFailed("provider timeout")

		require.NoError(t, err)
		assert.Equal(t, StatusFailed, n.Status())
		assert.Equal(t, "provider timeout", n.FailureReason())

		events := n.PullEvents()
		require.Len(t, events, 1)
		failed, ok := events[0].(NotificationFailed)
		require.True(t, ok)
		assert.Equal(t, "notification.failed", failed.EventName())
		assert.Equal(t, n.ID(), failed.AggregateID())
		assert.Equal(t, "provider timeout", failed.Reason)
	})

	t.Run("empty reason rejected", func(t *testing.T) {
		n := newQueued(t)

		err := n.MarkFailed("")

		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.Equal(t, StatusQueued, n.Status())
		assert.False(t, n.HasPendingEvents())
	})

	t.Run("already sent rejected", func(t *testing.T) {
		n := newQueued(t)
		require.NoError(t, n.MarkSent())
		n.PullEvents()

		err := n.MarkFailed("late failure")

		require.Error(t, err)
		assert.Equal(t, errs.KindConflict, errs.KindOf(err))
		assert.Equal(t, StatusSent, n.Status())
		assert.False(t, n.HasPendingEvents())
	})

	t.Run("retry then fail again", func(t *testing.T) {
		n := newQueued(t)
		require.NoError(t, n.MarkFailed("first"))
		require.NoError(t, n.MarkFailed("second"))

		assert.Equal(t, StatusFailed, n.Status())
		assert.Equal(t, "second", n.FailureReason())
		assert.Equal(t, []string{"notification.failed", "notification.failed"}, eventNames(n.PullEvents()))
	})
}

func TestReconstituteNotification(t *testing.T) {
	created := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	uid := userIDPtr(99)

	n := ReconstituteNotification(123, uid, TypePaymentFailed, ChannelPush, StatusFailed, "subj", "pl", 4, "declined", created)

	assert.Equal(t, uint64(123), n.ID())
	assert.Equal(t, uid, n.UserID())
	assert.Equal(t, TypePaymentFailed, n.Type())
	assert.Equal(t, ChannelPush, n.Channel())
	assert.Equal(t, StatusFailed, n.Status())
	assert.Equal(t, "subj", n.Subject())
	assert.Equal(t, "pl", n.Payload())
	assert.Equal(t, 4, n.Attempts())
	assert.Equal(t, "declined", n.FailureReason())
	assert.Equal(t, created, n.CreatedAt())
	assert.False(t, n.HasPendingEvents())
}
