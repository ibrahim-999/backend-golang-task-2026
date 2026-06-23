package gormrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/payment"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type paymentRepo struct {
	db *gorm.DB
}

func newPaymentRepo(db *gorm.DB) *paymentRepo {
	return &paymentRepo{db: db}
}

func (r *paymentRepo) Create(ctx context.Context, p *payment.Payment) error {
	m := paymentToModel(p)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	p.AssignID(m.ID)
	return nil
}

func (r *paymentRepo) Update(ctx context.Context, p *payment.Payment) error {
	m := paymentToModel(p)
	return r.db.WithContext(ctx).Save(&m).Error
}

func (r *paymentRepo) FindByID(ctx context.Context, id uint64) (*payment.Payment, error) {
	var m PaymentModel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("payment.not_found", "payment not found")
		}
		return nil, err
	}
	return paymentToDomain(m), nil
}

func (r *paymentRepo) FindByOrderID(ctx context.Context, orderID uint64) (*payment.Payment, error) {
	var m PaymentModel
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("payment.not_found", "payment not found for order")
		}
		return nil, err
	}
	return paymentToDomain(m), nil
}

func (r *paymentRepo) FindByIdempotencyKey(ctx context.Context, key string) (*payment.Payment, error) {
	var m PaymentModel
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("payment.not_found", "payment not found for idempotency key")
		}
		return nil, err
	}
	return paymentToDomain(m), nil
}

func paymentToModel(p *payment.Payment) PaymentModel {
	return PaymentModel{
		ID:             p.ID(),
		OrderID:        p.OrderID(),
		IdempotencyKey: p.IdempotencyKey(),
		Amount:         p.Amount().Amount(),
		Currency:       p.Amount().Currency(),
		Status:         string(p.Status()),
		Provider:       p.Provider(),
		ProviderRef:    p.ProviderRef(),
		Attempts:       p.Attempts(),
		FailureReason:  p.FailureReason(),
		CreatedAt:      p.CreatedAt(),
	}
}

func paymentToDomain(m PaymentModel) *payment.Payment {
	return payment.ReconstitutePayment(
		m.ID,
		m.OrderID,
		m.IdempotencyKey,
		shared.MustMoney(m.Amount, m.Currency),
		payment.Status(m.Status),
		m.Provider,
		m.ProviderRef,
		m.Attempts,
		m.FailureReason,
		m.CreatedAt,
	)
}

var _ ports.PaymentRepository = (*paymentRepo)(nil)
