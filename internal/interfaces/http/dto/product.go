package dto

import (
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
)

type CreateProductRequest struct {
	SKU          string `json:"sku" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	PriceAmount  int64  `json:"price_amount" binding:"gte=0"`
	Currency     string `json:"currency" binding:"required,len=3"`
	InitialStock int    `json:"initial_stock" binding:"gte=0"`
	ReorderLevel int    `json:"reorder_level" binding:"gte=0"`
}

type UpdateProductRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	PriceAmount *int64  `json:"price_amount" binding:"omitempty,gte=0"`
	Currency    *string `json:"currency" binding:"omitempty,len=3"`
	Active      *bool   `json:"active"`
}

type ProductResponse struct {
	ID          uint64 `json:"id"`
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceAmount int64  `json:"price_amount"`
	Currency    string `json:"currency"`
	Active      bool   `json:"active"`
}

type InventoryResponse struct {
	ProductID    uint64 `json:"product_id"`
	Available    int    `json:"available"`
	Reserved     int    `json:"reserved"`
	ReorderLevel int    `json:"reorder_level"`
}

func NewProductResponse(p *product.Product) ProductResponse {
	return ProductResponse{
		ID:          p.ID(),
		SKU:         p.SKU(),
		Name:        p.Name(),
		Description: p.Description(),
		PriceAmount: p.Price().Amount(),
		Currency:    p.Price().Currency(),
		Active:      p.Active(),
	}
}

func NewProductResponses(items []*product.Product) []ProductResponse {
	out := make([]ProductResponse, 0, len(items))
	for _, p := range items {
		out = append(out, NewProductResponse(p))
	}
	return out
}

func NewInventoryResponse(inv *inventory.Inventory) InventoryResponse {
	return InventoryResponse{
		ProductID:    inv.ProductID(),
		Available:    inv.Available(),
		Reserved:     inv.Reserved(),
		ReorderLevel: inv.ReorderLevel(),
	}
}
