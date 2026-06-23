package query

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
)

type LowStock struct {
	reads ports.RepositoryProvider
}

func NewLowStock(reads ports.RepositoryProvider) *LowStock {
	return &LowStock{reads: reads}
}

func (q *LowStock) Handle(ctx context.Context) ([]*inventory.Inventory, error) {
	return q.reads.Inventory().ListLowStock(ctx)
}
