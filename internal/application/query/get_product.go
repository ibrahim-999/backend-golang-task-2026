package query

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
)

type GetProduct struct {
	reads ports.RepositoryProvider
}

func NewGetProduct(reads ports.RepositoryProvider) GetProduct {
	return GetProduct{reads: reads}
}

func (q GetProduct) Handle(ctx context.Context, id uint64) (*product.Product, error) {
	return q.reads.Products().FindByID(ctx, id)
}
