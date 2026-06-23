package query

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
)

type ListUserOrders struct {
	reads ports.RepositoryProvider
}

func NewListUserOrders(reads ports.RepositoryProvider) *ListUserOrders {
	return &ListUserOrders{reads: reads}
}

func (q *ListUserOrders) Handle(ctx context.Context, userID uint64, page ports.Page) ([]*order.Order, int64, error) {
	return q.reads.Orders().ListByUser(ctx, userID, page)
}
