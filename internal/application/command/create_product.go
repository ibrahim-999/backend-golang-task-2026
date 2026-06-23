package command

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

type CreateProductInput struct {
	SKU          string
	Name         string
	Description  string
	PriceAmount  int64
	Currency     string
	InitialStock int
	ReorderLevel int
}

type CreateProduct struct {
	uow ports.UnitOfWork
}

func NewCreateProduct(uow ports.UnitOfWork) CreateProduct {
	return CreateProduct{uow: uow}
}

func (c CreateProduct) Handle(ctx context.Context, in CreateProductInput) (*product.Product, error) {
	price, err := shared.NewMoney(in.PriceAmount, in.Currency)
	if err != nil {
		return nil, err
	}

	p, err := product.NewProduct(in.SKU, in.Name, in.Description, price)
	if err != nil {
		return nil, err
	}

	err = c.uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		if err := rp.Products().Create(ctx, p); err != nil {
			return err
		}

		inv, err := inventory.NewInventory(p.ID(), in.InitialStock, in.ReorderLevel)
		if err != nil {
			return err
		}

		return rp.Inventory().Save(ctx, inv)
	})
	if err != nil {
		return nil, err
	}

	return p, nil
}
