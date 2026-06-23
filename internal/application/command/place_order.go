package command

import (
	"context"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/payment"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/concurrency"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type PlaceOrderItem struct {
	ProductID uint64
	Quantity  int
}

type PlaceOrderInput struct {
	UserID         uint64
	IdempotencyKey string
	Items          []PlaceOrderItem
}

type PlaceOrder struct {
	uow          ports.UnitOfWork
	reads        ports.RepositoryProvider
	gateway      ports.PaymentGateway
	bus          ports.EventBus
	pool         *concurrency.Pool
	maxAttempts  int
	retryBackoff time.Duration
}

func NewPlaceOrder(uow ports.UnitOfWork, reads ports.RepositoryProvider, gateway ports.PaymentGateway, bus ports.EventBus, pool *concurrency.Pool, maxAttempts int, retryBackoff time.Duration) *PlaceOrder {
	return &PlaceOrder{
		uow:          uow,
		reads:        reads,
		gateway:      gateway,
		bus:          bus,
		pool:         pool,
		maxAttempts:  maxAttempts,
		retryBackoff: retryBackoff,
	}
}

func (h *PlaceOrder) Handle(ctx context.Context, in PlaceOrderInput) (*order.Order, error) {
	if existing, err := h.reads.Orders().FindByIdempotencyKey(ctx, in.IdempotencyKey); err == nil && existing != nil {
		return existing, nil
	}

	items := make([]order.Item, 0, len(in.Items))
	for _, line := range in.Items {
		p, err := h.reads.Products().FindByID(ctx, line.ProductID)
		if err != nil {
			return nil, err
		}
		if !p.Active() {
			return nil, errs.Validation("order.inactive_product", "product is not available for purchase")
		}
		item, err := order.NewItem(p.ID(), p.SKU(), p.Name(), p.Price(), line.Quantity)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	ord, err := order.NewOrder(in.UserID, items, in.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	var pay *payment.Payment
	done := make(chan error, 1)
	submitErr := h.pool.Submit(func() {
		p, err := h.process(ctx, ord)
		pay = p
		done <- err
	})
	if submitErr != nil {
		return nil, errs.Unavailable("order.pool_unavailable", "order processing pool is not accepting work").WithCause(submitErr)
	}
	procErr := <-done

	events := ord.PullEvents()
	if pay != nil {
		events = append(events, pay.PullEvents()...)
	}
	_ = h.bus.Publish(ctx, events...)

	if procErr != nil {
		return nil, procErr
	}
	return ord, nil
}

func (h *PlaceOrder) process(ctx context.Context, ord *order.Order) (*payment.Payment, error) {
	txErr := h.uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		if err := rp.Orders().Create(ctx, ord); err != nil {
			return err
		}
		for _, item := range ord.Items() {
			ok, err := rp.Inventory().Reserve(ctx, item.ProductID(), item.Quantity())
			if err != nil {
				return err
			}
			if !ok {
				return errs.OutOfStock("order.out_of_stock", "insufficient stock to reserve order items")
			}
		}
		if err := ord.MarkReserved(); err != nil {
			return err
		}
		return rp.Orders().Update(ctx, ord)
	})
	if txErr != nil {
		return nil, txErr
	}

	pay, err := payment.NewPayment(ord.ID(), ord.IdempotencyKey(), ord.Total())
	if err != nil {
		return nil, err
	}

	attempts := h.maxAttempts
	if attempts < 1 {
		attempts = 1
	}
	var res ports.ChargeResult
	var chargeErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		_ = pay.RecordAttempt("mock")
		res, chargeErr = h.gateway.Charge(ctx, ports.ChargeRequest{
			IdempotencyKey: ord.IdempotencyKey(),
			Amount:         ord.Total(),
		})
		if chargeErr == nil || attempt == attempts {
			break
		}
		if h.retryBackoff > 0 {
			time.Sleep(h.retryBackoff * time.Duration(attempt))
		}
	}

	settleErr := h.uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		if chargeErr != nil {
			if err := pay.MarkFailed(chargeErr.Error()); err != nil {
				return err
			}
			if err := rp.Payments().Create(ctx, pay); err != nil {
				return err
			}
			for _, item := range ord.Items() {
				if err := rp.Inventory().Release(ctx, item.ProductID(), item.Quantity()); err != nil {
					return err
				}
			}
			if err := ord.Fail("payment failed"); err != nil {
				return err
			}
			return rp.Orders().Update(ctx, ord)
		}

		if err := pay.MarkSucceeded(res.ProviderRef); err != nil {
			return err
		}
		if err := rp.Payments().Create(ctx, pay); err != nil {
			return err
		}
		if err := ord.MarkPaid(); err != nil {
			return err
		}
		for _, item := range ord.Items() {
			if err := rp.Inventory().Commit(ctx, item.ProductID(), item.Quantity()); err != nil {
				return err
			}
		}
		if err := ord.MarkFulfilled(); err != nil {
			return err
		}
		return rp.Orders().Update(ctx, ord)
	})
	if settleErr != nil {
		return pay, settleErr
	}

	if chargeErr != nil {
		return pay, errs.PaymentFailed("order.payment_failed", "payment could not be completed").WithCause(chargeErr)
	}
	return pay, nil
}
