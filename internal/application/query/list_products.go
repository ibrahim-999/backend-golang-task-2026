package query

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
)

type ListProducts struct {
	reads ports.RepositoryProvider
}

func NewListProducts(reads ports.RepositoryProvider) ListProducts {
	return ListProducts{reads: reads}
}

func (q ListProducts) Handle(ctx context.Context, page ports.Page) ([]*product.Product, int64, error) {
	return q.reads.Products().List(ctx, page)
}
