package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/command"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/query"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/dto"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/httpx"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

const (
	productDefaultPageNumber = 1
	productDefaultPageSize   = 20
	productMaxPageSize       = 100
)

type Product struct {
	create       command.CreateProduct
	update       command.UpdateProduct
	list         query.ListProducts
	get          query.GetProduct
	getInventory query.GetInventory
}

func NewProduct(
	create command.CreateProduct,
	update command.UpdateProduct,
	list query.ListProducts,
	get query.GetProduct,
	getInventory query.GetInventory,
) *Product {
	return &Product{
		create:       create,
		update:       update,
		list:         list,
		get:          get,
		getInventory: getInventory,
	}
}

func (h *Product) List(c *gin.Context) {
	page := productPageFrom(c)

	items, total, err := h.list.Handle(c.Request.Context(), page)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.Page(c, dto.NewProductResponses(items), total, page.Number, page.Size)
}

func (h *Product) Get(c *gin.Context) {
	id, err := productIDFrom(c)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	p, err := h.get.Handle(c.Request.Context(), id)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, dto.NewProductResponse(p))
}

func (h *Product) Create(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, errs.Validation("product.invalid_request", err.Error()))
		return
	}

	p, err := h.create.Handle(c.Request.Context(), command.CreateProductInput{
		SKU:          req.SKU,
		Name:         req.Name,
		Description:  req.Description,
		PriceAmount:  req.PriceAmount,
		Currency:     req.Currency,
		InitialStock: req.InitialStock,
		ReorderLevel: req.ReorderLevel,
	})
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.Created(c, dto.NewProductResponse(p))
}

func (h *Product) Update(c *gin.Context) {
	id, err := productIDFrom(c)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, errs.Validation("product.invalid_request", err.Error()))
		return
	}

	p, err := h.update.Handle(c.Request.Context(), id, command.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		PriceAmount: req.PriceAmount,
		Currency:    req.Currency,
		Active:      req.Active,
	})
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, dto.NewProductResponse(p))
}

func (h *Product) Inventory(c *gin.Context) {
	id, err := productIDFrom(c)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	inv, err := h.getInventory.Handle(c.Request.Context(), id)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, dto.NewInventoryResponse(inv))
}

func productIDFrom(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errs.Validation("product.invalid_id", "invalid product id")
	}
	return id, nil
}

func productPageFrom(c *gin.Context) ports.Page {
	number := productDefaultPageNumber
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		number = v
	}

	size := productDefaultPageSize
	if v, err := strconv.Atoi(c.Query("size")); err == nil && v > 0 {
		size = v
	}
	if size > productMaxPageSize {
		size = productMaxPageSize
	}

	return ports.Page{Number: number, Size: size}
}
