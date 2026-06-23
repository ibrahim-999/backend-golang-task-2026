package gormrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type productRepo struct {
	db *gorm.DB
}

func newProductRepo(db *gorm.DB) *productRepo {
	return &productRepo{db: db}
}

func (r *productRepo) Create(ctx context.Context, p *product.Product) error {
	m := productToModel(p)
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *productRepo) Update(ctx context.Context, p *product.Product) error {
	m := productToModel(p)
	return r.db.WithContext(ctx).Save(&m).Error
}

func (r *productRepo) FindByID(ctx context.Context, id uint64) (*product.Product, error) {
	var m ProductModel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("product.not_found", "product not found")
		}
		return nil, err
	}
	return productToDomain(m), nil
}

func (r *productRepo) List(ctx context.Context, page ports.Page) ([]*product.Product, int64, error) {
	if page.Number <= 0 {
		page.Number = 1
	}
	if page.Size <= 0 {
		page.Size = 20
	}
	offset := (page.Number - 1) * page.Size

	var total int64
	if err := r.db.WithContext(ctx).Model(&ProductModel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ms []ProductModel
	if err := r.db.WithContext(ctx).Offset(offset).Limit(page.Size).Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	out := make([]*product.Product, 0, len(ms))
	for _, m := range ms {
		out = append(out, productToDomain(m))
	}
	return out, total, nil
}

func productToModel(p *product.Product) ProductModel {
	return ProductModel{
		ID:          p.ID(),
		SKU:         p.SKU(),
		Name:        p.Name(),
		Description: p.Description(),
		PriceAmount: p.Price().Amount(),
		Currency:    p.Price().Currency(),
		Active:      p.Active(),
		CreatedAt:   p.CreatedAt(),
	}
}

func productToDomain(m ProductModel) *product.Product {
	return product.ReconstituteProduct(
		m.ID,
		m.SKU,
		m.Name,
		m.Description,
		shared.MustMoney(m.PriceAmount, m.Currency),
		m.Active,
		m.CreatedAt,
	)
}

var _ ports.ProductRepository = (*productRepo)(nil)
