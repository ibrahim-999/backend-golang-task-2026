package command

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

type UpdateProductInput struct {
	Name        *string
	Description *string
	PriceAmount *int64
	Currency    *string
	Active      *bool
}

type UpdateProduct struct {
	uow ports.UnitOfWork
}

func NewUpdateProduct(uow ports.UnitOfWork) UpdateProduct {
	return UpdateProduct{uow: uow}
}

func (c UpdateProduct) Handle(ctx context.Context, id uint64, in UpdateProductInput) (*product.Product, error) {
	var result *product.Product

	err := c.uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		p, err := rp.Products().FindByID(ctx, id)
		if err != nil {
			return err
		}

		if in.Name != nil {
			if err := p.Rename(*in.Name); err != nil {
				return err
			}
		}

		if in.PriceAmount != nil {
			currency := p.Price().Currency()
			if in.Currency != nil {
				currency = *in.Currency
			}
			price, err := shared.NewMoney(*in.PriceAmount, currency)
			if err != nil {
				return err
			}
			p.Reprice(price)
		}

		if in.Active != nil {
			if *in.Active {
				p.Activate()
			} else {
				p.Deactivate()
			}
		}

		if err := rp.Products().Update(ctx, p); err != nil {
			return err
		}

		result = p
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
