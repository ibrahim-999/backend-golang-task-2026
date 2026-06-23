package query

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
)

type OrderStatus struct {
	reads ports.RepositoryProvider
}

func NewOrderStatus(reads ports.RepositoryProvider) *OrderStatus {
	return &OrderStatus{reads: reads}
}

func (q *OrderStatus) Handle(ctx context.Context, orderID uint64) (order.Status, error) {
	ord, err := q.reads.Orders().FindByID(ctx, orderID)
	if err != nil {
		return "", err
	}
	return ord.Status(), nil
}
