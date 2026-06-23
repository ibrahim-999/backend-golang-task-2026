package query

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
)

type GetInventory struct {
	reads ports.RepositoryProvider
}

func NewGetInventory(reads ports.RepositoryProvider) GetInventory {
	return GetInventory{reads: reads}
}

func (q GetInventory) Handle(ctx context.Context, productID uint64) (*inventory.Inventory, error) {
	return q.reads.Inventory().FindByProductID(ctx, productID)
}
