package query

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
)

type ListAllOrders struct {
	reads ports.RepositoryProvider
}

func NewListAllOrders(reads ports.RepositoryProvider) *ListAllOrders {
	return &ListAllOrders{reads: reads}
}

func (q *ListAllOrders) Handle(ctx context.Context, page ports.Page) ([]*order.Order, int64, error) {
	return q.reads.Orders().ListAll(ctx, page)
}
