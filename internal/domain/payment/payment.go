package payment

import (
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusRefunded  Status = "refunded"
)

type Payment struct {
	shared.AggregateRoot
	id             uint64
	orderID        uint64
	idempotencyKey string
	amount         shared.Money
	status         Status
	provider       string
	providerRef    string
	attempts       int
	failureReason  string
	createdAt      time.Time
}

func NewPayment(orderID uint64, idempotencyKey string, amount shared.Money) (*Payment, error) {
	if orderID == 0 {
		return nil, errs.Validation("payment.order_required", "order id is required")
	}
	if idempotencyKey == "" {
		return nil, errs.Validation("payment.idempotency_key_required", "idempotency key is required")
	}
	if amount.IsZero() {
		return nil, errs.Validation("payment.amount_positive", "amount must be greater than zero")
	}
	p := &Payment{
		orderID:        orderID,
		idempotencyKey: idempotencyKey,
		amount:         amount,
		status:         StatusPending,
		createdAt:      time.Now().UTC(),
	}
	p.Record(PaymentInitiated{PaymentID: p.id, OrderID: p.orderID})
	return p, nil
}

func ReconstitutePayment(
	id, orderID uint64,
	idempotencyKey string,
	amount shared.Money,
	status Status,
	provider, providerRef string,
	attempts int,
	failureReason string,
	createdAt time.Time,
) *Payment {
	return &Payment{
		id:             id,
		orderID:        orderID,
		idempotencyKey: idempotencyKey,
		amount:         amount,
		status:         status,
		provider:       provider,
		providerRef:    providerRef,
		attempts:       attempts,
		failureReason:  failureReason,
		createdAt:      createdAt,
	}
}

func (p *Payment) AssignID(id uint64) {
	if p.id == 0 {
		p.id = id
	}
}

func (p *Payment) ID() uint64             { return p.id }
func (p *Payment) OrderID() uint64        { return p.orderID }
func (p *Payment) IdempotencyKey() string { return p.idempotencyKey }
func (p *Payment) Amount() shared.Money   { return p.amount }
func (p *Payment) Status() Status         { return p.status }
func (p *Payment) Provider() string       { return p.provider }
func (p *Payment) ProviderRef() string    { return p.providerRef }
func (p *Payment) Attempts() int          { return p.attempts }
func (p *Payment) FailureReason() string  { return p.failureReason }
func (p *Payment) CreatedAt() time.Time   { return p.createdAt }

func (p *Payment) RecordAttempt(provider string) error {
	if provider == "" {
		return errs.Validation("payment.provider_required", "provider is required")
	}
	p.attempts++
	p.provider = provider
	return nil
}

func (p *Payment) MarkSucceeded(providerRef string) error {
	if p.status != StatusPending {
		return errs.Conflict("payment.not_pending", "payment is not pending")
	}
	if providerRef == "" {
		return errs.Validation("payment.provider_ref_required", "provider reference is required")
	}
	p.status = StatusSucceeded
	p.providerRef = providerRef
	p.failureReason = ""
	p.Record(PaymentSucceeded{PaymentID: p.id, OrderID: p.orderID})
	return nil
}

func (p *Payment) MarkFailed(reason string) error {
	if p.status != StatusPending {
		return errs.Conflict("payment.not_pending", "payment is not pending")
	}
	if reason == "" {
		return errs.Validation("payment.reason_required", "failure reason is required")
	}
	p.status = StatusFailed
	p.failureReason = reason
	p.Record(PaymentFailed{PaymentID: p.id, OrderID: p.orderID, Reason: reason})
	return nil
}

func (p *Payment) Refund() error {
	if p.status != StatusSucceeded {
		return errs.Conflict("payment.not_succeeded", "only succeeded payments can be refunded")
	}
	p.status = StatusRefunded
	p.Record(PaymentRefunded{PaymentID: p.id, OrderID: p.orderID})
	return nil
}
