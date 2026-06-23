package query

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
)

type GetOrder struct {
	reads ports.RepositoryProvider
}

func NewGetOrder(reads ports.RepositoryProvider) *GetOrder {
	return &GetOrder{reads: reads}
}

func (q *GetOrder) Handle(ctx context.Context, orderID uint64) (*order.Order, error) {
	return q.reads.Orders().FindByID(ctx, orderID)
}
