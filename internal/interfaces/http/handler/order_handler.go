package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/command"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/query"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/httpx"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/middleware"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

const (
	idempotencyKeyHeader   = "Idempotency-Key"
	roleAdmin              = "admin"
	orderDefaultPageNumber = 1
	orderDefaultPageSize   = 20
	orderMaxPageSize       = 100
)

type PlaceOrderItemRequest struct {
	ProductID uint64 `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
}

type PlaceOrderRequest struct {
	Items []PlaceOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

type OrderItemResponse struct {
	ProductID uint64 `json:"product_id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	UnitPrice int64  `json:"unit_price"`
	Quantity  int    `json:"quantity"`
	Subtotal  int64  `json:"subtotal"`
}

type OrderResponse struct {
	ID          uint64              `json:"id"`
	Status      string              `json:"status"`
	TotalAmount int64               `json:"total_amount"`
	Currency    string              `json:"currency"`
	Items       []OrderItemResponse `json:"items"`
	CreatedAt   time.Time           `json:"created_at"`
}

type OrderStatusResponse struct {
	ID     uint64 `json:"id"`
	Status string `json:"status"`
}

type Order struct {
	place      *command.PlaceOrder
	cancel     *command.CancelOrder
	get        *query.GetOrder
	status     *query.OrderStatus
	listByUser *query.ListUserOrders
}

func NewOrder(place *command.PlaceOrder, cancel *command.CancelOrder, get *query.GetOrder, status *query.OrderStatus, listByUser *query.ListUserOrders) *Order {
	return &Order{
		place:      place,
		cancel:     cancel,
		get:        get,
		status:     status,
		listByUser: listByUser,
	}
}

func (h *Order) Place(c *gin.Context) {
	userID, ok := middleware.UserIDFrom(c)
	if !ok {
		httpx.Error(c, errs.Unauthorized("unauthorized", "authentication required"))
		return
	}

	var req PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, errs.Validation("order.invalid_request", err.Error()))
		return
	}

	key := c.GetHeader(idempotencyKeyHeader)
	if key == "" {
		key = uuid.NewString()
	}

	items := make([]command.PlaceOrderItem, 0, len(req.Items))
	for _, line := range req.Items {
		items = append(items, command.PlaceOrderItem{ProductID: line.ProductID, Quantity: line.Quantity})
	}

	ord, err := h.place.Handle(c.Request.Context(), command.PlaceOrderInput{
		UserID:         userID,
		IdempotencyKey: key,
		Items:          items,
	})
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.Created(c, toUserOrderResponse(ord))
}

func (h *Order) List(c *gin.Context) {
	userID, ok := middleware.UserIDFrom(c)
	if !ok {
		httpx.Error(c, errs.Unauthorized("unauthorized", "authentication required"))
		return
	}

	page := pageFrom(c)
	orders, total, err := h.listByUser.Handle(c.Request.Context(), userID, page)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	items := make([]OrderResponse, 0, len(orders))
	for _, ord := range orders {
		items = append(items, toUserOrderResponse(ord))
	}

	httpx.Page(c, items, total, page.Number, page.Size)
}

func (h *Order) Get(c *gin.Context) {
	userID, ok := middleware.UserIDFrom(c)
	if !ok {
		httpx.Error(c, errs.Unauthorized("unauthorized", "authentication required"))
		return
	}

	id, err := orderIDFrom(c)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	ord, err := h.get.Handle(c.Request.Context(), id)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	if ord.UserID() != userID && middleware.RoleFrom(c) != roleAdmin {
		httpx.Error(c, errs.Forbidden("order.forbidden", "you do not have access to this order"))
		return
	}

	httpx.OK(c, toUserOrderResponse(ord))
}

func (h *Order) Cancel(c *gin.Context) {
	userID, ok := middleware.UserIDFrom(c)
	if !ok {
		httpx.Error(c, errs.Unauthorized("unauthorized", "authentication required"))
		return
	}

	id, err := orderIDFrom(c)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	ord, err := h.get.Handle(c.Request.Context(), id)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	if ord.UserID() != userID && middleware.RoleFrom(c) != roleAdmin {
		httpx.Error(c, errs.Forbidden("order.forbidden", "you do not have access to this order"))
		return
	}

	cancelled, err := h.cancel.Handle(c.Request.Context(), command.CancelOrderInput{
		OrderID: id,
		Reason:  "cancelled by user",
	})
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, toUserOrderResponse(cancelled))
}

func (h *Order) Status(c *gin.Context) {
	userID, ok := middleware.UserIDFrom(c)
	if !ok {
		httpx.Error(c, errs.Unauthorized("unauthorized", "authentication required"))
		return
	}

	id, err := orderIDFrom(c)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	ord, err := h.get.Handle(c.Request.Context(), id)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	if ord.UserID() != userID && middleware.RoleFrom(c) != roleAdmin {
		httpx.Error(c, errs.Forbidden("order.forbidden", "you do not have access to this order"))
		return
	}

	httpx.OK(c, OrderStatusResponse{ID: ord.ID(), Status: string(ord.Status())})
}

func orderIDFrom(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errs.Validation("order.invalid_id", "order id must be a positive integer")
	}
	return id, nil
}

func pageFrom(c *gin.Context) ports.Page {
	number := orderDefaultPageNumber
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		number = v
	}
	size := orderDefaultPageSize
	if v, err := strconv.Atoi(c.Query("size")); err == nil && v > 0 {
		size = v
	}
	if size > orderMaxPageSize {
		size = orderMaxPageSize
	}
	return ports.Page{Number: number, Size: size}
}

func toUserOrderResponse(ord *order.Order) OrderResponse {
	items := make([]OrderItemResponse, 0, len(ord.Items()))
	for _, item := range ord.Items() {
		items = append(items, OrderItemResponse{
			ProductID: item.ProductID(),
			SKU:       item.SKU(),
			Name:      item.Name(),
			UnitPrice: item.UnitPrice().Amount(),
			Quantity:  item.Quantity(),
			Subtotal:  item.Subtotal().Amount(),
		})
	}
	return OrderResponse{
		ID:          ord.ID(),
		Status:      string(ord.Status()),
		TotalAmount: ord.Total().Amount(),
		Currency:    ord.Total().Currency(),
		Items:       items,
		CreatedAt:   ord.CreatedAt(),
	}
}
