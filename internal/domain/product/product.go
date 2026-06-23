package product

import (
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type Product struct {
	shared.AggregateRoot
	id          uint64
	sku         string
	name        string
	description string
	price       shared.Money
	active      bool
	createdAt   time.Time
}

func NewProduct(sku, name, description string, price shared.Money) (*Product, error) {
	if sku == "" {
		return nil, errs.Validation("product.sku_required", "sku cannot be empty")
	}
	if name == "" {
		return nil, errs.Validation("product.name_required", "name cannot be empty")
	}

	p := &Product{
		sku:         sku,
		name:        name,
		description: description,
		price:       price,
		active:      true,
		createdAt:   time.Now().UTC(),
	}
	p.Record(ProductCreated{ProductID: p.id, SKU: p.sku})
	return p, nil
}

func ReconstituteProduct(id uint64, sku, name, description string, price shared.Money, active bool, createdAt time.Time) *Product {
	return &Product{
		id:          id,
		sku:         sku,
		name:        name,
		description: description,
		price:       price,
		active:      active,
		createdAt:   createdAt,
	}
}

func (p *Product) Rename(name string) error {
	if name == "" {
		return errs.Validation("product.name_required", "name cannot be empty")
	}
	p.name = name
	return nil
}

func (p *Product) Reprice(price shared.Money) {
	p.price = price
	p.Record(ProductRepriced{ProductID: p.id})
}

func (p *Product) Activate() {
	p.active = true
}

func (p *Product) Deactivate() {
	p.active = false
}

func (p *Product) ID() uint64           { return p.id }
func (p *Product) SKU() string          { return p.sku }
func (p *Product) Name() string         { return p.name }
func (p *Product) Description() string  { return p.description }
func (p *Product) Price() shared.Money  { return p.price }
func (p *Product) Active() bool         { return p.active }
func (p *Product) CreatedAt() time.Time { return p.createdAt }
