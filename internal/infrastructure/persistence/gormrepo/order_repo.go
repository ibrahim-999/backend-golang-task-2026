package gormrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type orderRepo struct {
	db *gorm.DB
}

func newOrderRepo(db *gorm.DB) *orderRepo {
	return &orderRepo{db: db}
}

func (r *orderRepo) Create(ctx context.Context, o *order.Order) error {
	m := orderToModel(o)
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *orderRepo) Update(ctx context.Context, o *order.Order) error {
	m := orderToModel(o)
	return r.db.WithContext(ctx).Save(&m).Error
}

func (r *orderRepo) FindByID(ctx context.Context, id uint64) (*order.Order, error) {
	var m OrderModel
	if err := r.db.WithContext(ctx).Preload("Items").First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("order.not_found", "order not found")
		}
		return nil, err
	}
	return orderToDomain(m)
}

func (r *orderRepo) FindByIdempotencyKey(ctx context.Context, key string) (*order.Order, error) {
	var m OrderModel
	if err := r.db.WithContext(ctx).Preload("Items").Where("idempotency_key = ?", key).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("order.not_found", "order not found")
		}
		return nil, err
	}
	return orderToDomain(m)
}

func (r *orderRepo) ListByUser(ctx context.Context, userID uint64, page ports.Page) ([]*order.Order, int64, error) {
	number, size := normalizeOrderPage(page)
	offset := (number - 1) * size

	var total int64
	if err := r.db.WithContext(ctx).Model(&OrderModel{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ms []OrderModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Where("user_id = ?", userID).
		Offset(offset).
		Limit(size).
		Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	out, err := ordersToDomain(ms)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *orderRepo) ListAll(ctx context.Context, page ports.Page) ([]*order.Order, int64, error) {
	number, size := normalizeOrderPage(page)
	offset := (number - 1) * size

	var total int64
	if err := r.db.WithContext(ctx).Model(&OrderModel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ms []OrderModel
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Offset(offset).
		Limit(size).
		Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	out, err := ordersToDomain(ms)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func normalizeOrderPage(page ports.Page) (int, int) {
	number := page.Number
	size := page.Size
	if number <= 0 {
		number = 1
	}
	if size <= 0 {
		size = 20
	}
	return number, size
}

func orderToModel(o *order.Order) OrderModel {
	items := make([]OrderItemModel, 0, len(o.Items()))
	for _, item := range o.Items() {
		items = append(items, OrderItemModel{
			ProductID:       item.ProductID(),
			SKU:             item.SKU(),
			Name:            item.Name(),
			UnitPriceAmount: item.UnitPrice().Amount(),
			Currency:        item.UnitPrice().Currency(),
			Quantity:        item.Quantity(),
		})
	}
	return OrderModel{
		ID:             o.ID(),
		UserID:         o.UserID(),
		Status:         string(o.Status()),
		TotalAmount:    o.Total().Amount(),
		Currency:       o.Total().Currency(),
		IdempotencyKey: o.IdempotencyKey(),
		FailureReason:  o.FailureReason(),
		Items:          items,
	}
}

func orderToDomain(m OrderModel) (*order.Order, error) {
	items := make([]order.Item, 0, len(m.Items))
	for _, im := range m.Items {
		item, err := order.NewItem(im.ProductID, im.SKU, im.Name, shared.MustMoney(im.UnitPriceAmount, im.Currency), im.Quantity)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return order.ReconstituteOrder(
		m.ID,
		m.UserID,
		order.Status(m.Status),
		items,
		shared.MustMoney(m.TotalAmount, m.Currency),
		m.IdempotencyKey,
		m.FailureReason,
		m.CreatedAt,
	), nil
}

func ordersToDomain(ms []OrderModel) ([]*order.Order, error) {
	out := make([]*order.Order, 0, len(ms))
	for _, m := range ms {
		o, err := orderToDomain(m)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

var _ ports.OrderRepository = (*orderRepo)(nil)
