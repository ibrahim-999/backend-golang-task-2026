package inventory

import (
	"testing"

	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInventory(t *testing.T) {
	tests := []struct {
		name         string
		productID    uint64
		available    int
		reorderLevel int
		wantKind     errs.Kind
		wantErr      bool
	}{
		{name: "valid", productID: 7, available: 10, reorderLevel: 3, wantErr: false},
		{name: "valid zero available", productID: 7, available: 0, reorderLevel: 0, wantErr: false},
		{name: "missing product", productID: 0, available: 10, reorderLevel: 3, wantErr: true, wantKind: errs.KindValidation},
		{name: "negative available", productID: 7, available: -1, reorderLevel: 3, wantErr: true, wantKind: errs.KindValidation},
		{name: "negative reorder level", productID: 7, available: 10, reorderLevel: -1, wantErr: true, wantKind: errs.KindValidation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, err := NewInventory(tt.productID, tt.available, tt.reorderLevel)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantKind, errs.KindOf(err))
				assert.Nil(t, inv)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, inv)
			assert.Equal(t, uint64(0), inv.ID())
			assert.Equal(t, tt.productID, inv.ProductID())
			assert.Equal(t, tt.available, inv.Available())
			assert.Equal(t, 0, inv.Reserved())
			assert.Equal(t, tt.reorderLevel, inv.ReorderLevel())
			assert.Equal(t, 0, inv.Version())
			assert.False(t, inv.HasPendingEvents())
		})
	}
}

func TestReconstituteInventory(t *testing.T) {
	inv := ReconstituteInventory(42, 7, 5, 2, 3, 9)
	assert.Equal(t, uint64(42), inv.ID())
	assert.Equal(t, uint64(7), inv.ProductID())
	assert.Equal(t, 5, inv.Available())
	assert.Equal(t, 2, inv.Reserved())
	assert.Equal(t, 3, inv.ReorderLevel())
	assert.Equal(t, 9, inv.Version())
	assert.False(t, inv.HasPendingEvents())
}

func TestReserve(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 10, 0, 2, 4)
		require.NoError(t, inv.Reserve(3))
		assert.Equal(t, 7, inv.Available())
		assert.Equal(t, 3, inv.Reserved())
		assert.Equal(t, 5, inv.Version())
		events := inv.PullEvents()
		require.Len(t, events, 1)
		ev, ok := events[0].(InventoryReserved)
		require.True(t, ok)
		assert.Equal(t, "inventory.reserved", ev.EventName())
		assert.Equal(t, uint64(7), ev.AggregateID())
		assert.Equal(t, 3, ev.Quantity)
	})

	t.Run("exact available", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 5, 0, 2, 0)
		require.NoError(t, inv.Reserve(5))
		assert.Equal(t, 0, inv.Available())
		assert.Equal(t, 5, inv.Reserved())
	})

	t.Run("non positive quantity", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 10, 0, 2, 0)
		err := inv.Reserve(0)
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.Equal(t, 10, inv.Available())
		assert.Equal(t, 0, inv.Version())
		assert.False(t, inv.HasPendingEvents())
	})

	t.Run("insufficient stock", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 2, 0, 2, 0)
		err := inv.Reserve(3)
		require.Error(t, err)
		assert.Equal(t, errs.KindOutOfStock, errs.KindOf(err))
		assert.Equal(t, 2, inv.Available())
		assert.Equal(t, 0, inv.Reserved())
		assert.Equal(t, 0, inv.Version())
		assert.False(t, inv.HasPendingEvents())
	})
}

func TestRelease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 4, 6, 2, 3)
		require.NoError(t, inv.Release(2))
		assert.Equal(t, 6, inv.Available())
		assert.Equal(t, 4, inv.Reserved())
		assert.Equal(t, 3, inv.Version())
		events := inv.PullEvents()
		require.Len(t, events, 1)
		ev, ok := events[0].(InventoryReleased)
		require.True(t, ok)
		assert.Equal(t, "inventory.released", ev.EventName())
		assert.Equal(t, uint64(7), ev.AggregateID())
		assert.Equal(t, 2, ev.Quantity)
	})

	t.Run("non positive quantity", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 4, 6, 2, 0)
		err := inv.Release(-1)
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.False(t, inv.HasPendingEvents())
	})

	t.Run("more than reserved", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 4, 1, 2, 0)
		err := inv.Release(2)
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.Equal(t, 4, inv.Available())
		assert.Equal(t, 1, inv.Reserved())
		assert.False(t, inv.HasPendingEvents())
	})
}

func TestCommit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 4, 6, 2, 3)
		require.NoError(t, inv.Commit(2))
		assert.Equal(t, 4, inv.Available())
		assert.Equal(t, 4, inv.Reserved())
		assert.Equal(t, 3, inv.Version())
		events := inv.PullEvents()
		require.Len(t, events, 1)
		ev, ok := events[0].(InventoryCommitted)
		require.True(t, ok)
		assert.Equal(t, "inventory.committed", ev.EventName())
		assert.Equal(t, uint64(7), ev.AggregateID())
		assert.Equal(t, 2, ev.Quantity)
	})

	t.Run("non positive quantity", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 4, 6, 2, 0)
		err := inv.Commit(0)
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.False(t, inv.HasPendingEvents())
	})

	t.Run("more than reserved", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 4, 1, 2, 0)
		err := inv.Commit(2)
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.Equal(t, 4, inv.Available())
		assert.Equal(t, 1, inv.Reserved())
		assert.False(t, inv.HasPendingEvents())
	})
}

func TestRestock(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 4, 6, 2, 3)
		require.NoError(t, inv.Restock(5))
		assert.Equal(t, 9, inv.Available())
		assert.Equal(t, 6, inv.Reserved())
		assert.Equal(t, 4, inv.Version())
		events := inv.PullEvents()
		require.Len(t, events, 1)
		ev, ok := events[0].(InventoryRestocked)
		require.True(t, ok)
		assert.Equal(t, "inventory.restocked", ev.EventName())
		assert.Equal(t, uint64(7), ev.AggregateID())
		assert.Equal(t, 5, ev.Quantity)
	})

	t.Run("non positive quantity", func(t *testing.T) {
		inv := ReconstituteInventory(1, 7, 4, 6, 2, 0)
		err := inv.Restock(0)
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
		assert.Equal(t, 4, inv.Available())
		assert.Equal(t, 0, inv.Version())
		assert.False(t, inv.HasPendingEvents())
	})
}

func TestIsLow(t *testing.T) {
	tests := []struct {
		name         string
		available    int
		reorderLevel int
		want         bool
	}{
		{name: "below reorder", available: 1, reorderLevel: 3, want: true},
		{name: "at reorder", available: 3, reorderLevel: 3, want: true},
		{name: "above reorder", available: 4, reorderLevel: 3, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := ReconstituteInventory(1, 7, tt.available, 0, tt.reorderLevel, 0)
			assert.Equal(t, tt.want, inv.IsLow())
		})
	}
}

func TestReserveThenCommitLifecycle(t *testing.T) {
	inv, err := NewInventory(7, 10, 2)
	require.NoError(t, err)

	require.NoError(t, inv.Reserve(4))
	require.NoError(t, inv.Commit(3))
	require.NoError(t, inv.Release(1))

	assert.Equal(t, 7, inv.Available())
	assert.Equal(t, 0, inv.Reserved())

	events := inv.PullEvents()
	require.Len(t, events, 3)
	assert.Equal(t, "inventory.reserved", events[0].EventName())
	assert.Equal(t, "inventory.committed", events[1].EventName())
	assert.Equal(t, "inventory.released", events[2].EventName())
	assert.False(t, inv.HasPendingEvents())
}
