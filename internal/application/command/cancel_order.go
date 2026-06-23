package command

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
)

type CancelOrderInput struct {
	OrderID uint64
	Reason  string
}

type CancelOrder struct {
	uow   ports.UnitOfWork
	reads ports.RepositoryProvider
	bus   ports.EventBus
}

func NewCancelOrder(uow ports.UnitOfWork, reads ports.RepositoryProvider, bus ports.EventBus) *CancelOrder {
	return &CancelOrder{uow: uow, reads: reads, bus: bus}
}

func (h *CancelOrder) Handle(ctx context.Context, in CancelOrderInput) (*order.Order, error) {
	ord, err := h.reads.Orders().FindByID(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}

	wasReserved := ord.Status() == order.StatusReserved
	wasPaid := ord.Status() == order.StatusPaid

	txErr := h.uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		if wasReserved || wasPaid {
			for _, item := range ord.Items() {
				if err := rp.Inventory().Release(ctx, item.ProductID(), item.Quantity()); err != nil {
					return err
				}
			}
		}
		if wasPaid {
			pay, err := rp.Payments().FindByOrderID(ctx, ord.ID())
			if err != nil {
				return err
			}
			if err := pay.Refund(); err != nil {
				return err
			}
			if err := rp.Payments().Update(ctx, pay); err != nil {
				return err
			}
		}
		if err := ord.Cancel(in.Reason); err != nil {
			return err
		}
		return rp.Orders().Update(ctx, ord)
	})
	if txErr != nil {
		return nil, txErr
	}

	_ = h.bus.Publish(ctx, ord.PullEvents()...)
	return ord, nil
}
