package command

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type UpdateOrderStatus struct {
	uow ports.UnitOfWork
}

func NewUpdateOrderStatus(uow ports.UnitOfWork) *UpdateOrderStatus {
	return &UpdateOrderStatus{uow: uow}
}

func (c *UpdateOrderStatus) Handle(ctx context.Context, orderID uint64, target order.Status) error {
	return c.uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		o, err := rp.Orders().FindByID(ctx, orderID)
		if err != nil {
			return err
		}

		if err := transition(o, target); err != nil {
			return err
		}

		return rp.Orders().Update(ctx, o)
	})
}

func transition(o *order.Order, target order.Status) error {
	switch target {
	case order.StatusReserved:
		return o.MarkReserved()
	case order.StatusPaid:
		return o.MarkPaid()
	case order.StatusFulfilled:
		return o.MarkFulfilled()
	case order.StatusCancelled:
		return o.Cancel("status updated by administrator")
	case order.StatusFailed:
		return o.Fail("status updated by administrator")
	default:
		return errs.Validation("order.unsupported_target_status", "unsupported target status")
	}
}
