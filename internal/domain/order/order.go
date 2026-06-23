package order

import (
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusReserved  Status = "reserved"
	StatusPaid      Status = "paid"
	StatusFulfilled Status = "fulfilled"
	StatusCancelled Status = "cancelled"
	StatusFailed    Status = "failed"
)

type Item struct {
	productID uint64
	sku       string
	name      string
	unitPrice shared.Money
	quantity  int
}

func NewItem(productID uint64, sku, name string, unitPrice shared.Money, quantity int) (Item, error) {
	if quantity <= 0 {
		return Item{}, errs.Validation("order.item_quantity", "item quantity must be greater than zero")
	}
	return Item{
		productID: productID,
		sku:       sku,
		name:      name,
		unitPrice: unitPrice,
		quantity:  quantity,
	}, nil
}

func (i Item) ProductID() uint64       { return i.productID }
func (i Item) SKU() string             { return i.sku }
func (i Item) Name() string            { return i.name }
func (i Item) UnitPrice() shared.Money { return i.unitPrice }
func (i Item) Quantity() int           { return i.quantity }
func (i Item) Subtotal() shared.Money  { return i.unitPrice.Mul(int64(i.quantity)) }

type Order struct {
	shared.AggregateRoot
	id             uint64
	userID         uint64
	status         Status
	items          []Item
	total          shared.Money
	idempotencyKey string
	failureReason  string
	createdAt      time.Time
}

func NewOrder(userID uint64, items []Item, idempotencyKey string) (*Order, error) {
	if len(items) < 1 {
		return nil, errs.Validation("order.no_items", "order must contain at least one item")
	}
	if idempotencyKey == "" {
		return nil, errs.Validation("order.idempotency_key", "idempotency key is required")
	}

	currency := items[0].UnitPrice().Currency()
	total := shared.ZeroMoney(currency)
	for _, item := range items {
		if item.UnitPrice().Currency() != currency {
			return nil, errs.Validation("order.currency_mismatch", "all items must share the same currency")
		}
		sum, err := total.Add(item.Subtotal())
		if err != nil {
			return nil, errs.Validation("order.currency_mismatch", "all items must share the same currency")
		}
		total = sum
	}

	o := &Order{
		userID:         userID,
		status:         StatusPending,
		items:          items,
		total:          total,
		idempotencyKey: idempotencyKey,
		createdAt:      time.Now().UTC(),
	}
	o.Record(OrderPlaced{
		OrderID:     o.id,
		UserID:      userID,
		TotalAmount: total.Amount(),
		Currency:    total.Currency(),
	})
	return o, nil
}

func ReconstituteOrder(id, userID uint64, status Status, items []Item, total shared.Money, idempotencyKey, failureReason string, createdAt time.Time) *Order {
	return &Order{
		id:             id,
		userID:         userID,
		status:         status,
		items:          items,
		total:          total,
		idempotencyKey: idempotencyKey,
		failureReason:  failureReason,
		createdAt:      createdAt,
	}
}

func (o *Order) AssignID(id uint64) {
	if o.id == 0 {
		o.id = id
	}
}

func (o *Order) ID() uint64             { return o.id }
func (o *Order) UserID() uint64         { return o.userID }
func (o *Order) Status() Status         { return o.status }
func (o *Order) Items() []Item          { return o.items }
func (o *Order) Total() shared.Money    { return o.total }
func (o *Order) IdempotencyKey() string { return o.idempotencyKey }
func (o *Order) FailureReason() string  { return o.failureReason }
func (o *Order) CreatedAt() time.Time   { return o.createdAt }

func (o *Order) IsTerminal() bool {
	switch o.status {
	case StatusFulfilled, StatusCancelled, StatusFailed:
		return true
	default:
		return false
	}
}

func (o *Order) CanCancel() bool {
	switch o.status {
	case StatusPending, StatusReserved, StatusPaid:
		return true
	default:
		return false
	}
}

func (o *Order) MarkReserved() error {
	if o.status != StatusPending {
		return errs.Conflict("order.invalid_transition", "order can only be reserved from pending")
	}
	o.status = StatusReserved
	o.Record(OrderReserved{OrderID: o.id})
	return nil
}

func (o *Order) MarkPaid() error {
	if o.status != StatusReserved {
		return errs.Conflict("order.invalid_transition", "order can only be paid from reserved")
	}
	o.status = StatusPaid
	o.Record(OrderPaid{OrderID: o.id})
	return nil
}

func (o *Order) MarkFulfilled() error {
	if o.status != StatusPaid {
		return errs.Conflict("order.invalid_transition", "order can only be fulfilled from paid")
	}
	o.status = StatusFulfilled
	o.Record(OrderFulfilled{OrderID: o.id})
	return nil
}

func (o *Order) Cancel(reason string) error {
	if !o.CanCancel() {
		return errs.Conflict("order.invalid_transition", "order cannot be cancelled from its current status")
	}
	o.status = StatusCancelled
	o.Record(OrderCancelled{OrderID: o.id, Reason: reason})
	return nil
}

func (o *Order) Fail(reason string) error {
	if o.IsTerminal() {
		return errs.Conflict("order.invalid_transition", "order cannot be failed from a terminal status")
	}
	o.status = StatusFailed
	o.failureReason = reason
	o.Record(OrderFailed{OrderID: o.id, Reason: reason})
	return nil
}
