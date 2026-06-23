package inventory

import (
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type Inventory struct {
	shared.AggregateRoot
	id           uint64
	productID    uint64
	available    int
	reserved     int
	reorderLevel int
	version      int
}

func NewInventory(productID uint64, available, reorderLevel int) (*Inventory, error) {
	if productID == 0 {
		return nil, errs.Validation("inventory.invalid_product", "product id is required")
	}
	if available < 0 {
		return nil, errs.Validation("inventory.invalid_available", "available must not be negative")
	}
	if reorderLevel < 0 {
		return nil, errs.Validation("inventory.invalid_reorder_level", "reorder level must not be negative")
	}
	return &Inventory{
		productID:    productID,
		available:    available,
		reserved:     0,
		reorderLevel: reorderLevel,
		version:      0,
	}, nil
}

func ReconstituteInventory(id, productID uint64, available, reserved, reorderLevel, version int) *Inventory {
	return &Inventory{
		id:           id,
		productID:    productID,
		available:    available,
		reserved:     reserved,
		reorderLevel: reorderLevel,
		version:      version,
	}
}

func (i *Inventory) Reserve(qty int) error {
	if qty <= 0 {
		return errs.Validation("inventory.invalid_quantity", "quantity must be positive")
	}
	if i.available < qty {
		return errs.OutOfStock("inventory.insufficient_stock", "not enough available stock to reserve")
	}
	i.available -= qty
	i.reserved += qty
	i.version++
	i.Record(InventoryReserved{ProductID: i.productID, Quantity: qty})
	return nil
}

func (i *Inventory) Release(qty int) error {
	if qty <= 0 {
		return errs.Validation("inventory.invalid_quantity", "quantity must be positive")
	}
	if i.reserved < qty {
		return errs.Validation("inventory.insufficient_reserved", "not enough reserved stock to release")
	}
	i.reserved -= qty
	i.available += qty
	i.Record(InventoryReleased{ProductID: i.productID, Quantity: qty})
	return nil
}

func (i *Inventory) Commit(qty int) error {
	if qty <= 0 {
		return errs.Validation("inventory.invalid_quantity", "quantity must be positive")
	}
	if i.reserved < qty {
		return errs.Validation("inventory.insufficient_reserved", "not enough reserved stock to commit")
	}
	i.reserved -= qty
	i.Record(InventoryCommitted{ProductID: i.productID, Quantity: qty})
	return nil
}

func (i *Inventory) Restock(qty int) error {
	if qty <= 0 {
		return errs.Validation("inventory.invalid_quantity", "quantity must be positive")
	}
	i.available += qty
	i.version++
	i.Record(InventoryRestocked{ProductID: i.productID, Quantity: qty})
	return nil
}

func (i *Inventory) IsLow() bool {
	return i.available <= i.reorderLevel
}

func (i *Inventory) ID() uint64 {
	return i.id
}

func (i *Inventory) ProductID() uint64 {
	return i.productID
}

func (i *Inventory) Available() int {
	return i.available
}

func (i *Inventory) Reserved() int {
	return i.reserved
}

func (i *Inventory) ReorderLevel() int {
	return i.reorderLevel
}

func (i *Inventory) Version() int {
	return i.version
}
