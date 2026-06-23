package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/command"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/query"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/httpx"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

const (
	defaultPageNumber = 1
	defaultPageSize   = 20
	maxPageSize       = 100
)

type Admin struct {
	listAllOrders     *query.ListAllOrders
	updateOrderStatus *command.UpdateOrderStatus
	dailyReport       *query.DailyReport
	lowStock          *query.LowStock
}

func NewAdmin(
	listAllOrders *query.ListAllOrders,
	updateOrderStatus *command.UpdateOrderStatus,
	dailyReport *query.DailyReport,
	lowStock *query.LowStock,
) *Admin {
	return &Admin{
		listAllOrders:     listAllOrders,
		updateOrderStatus: updateOrderStatus,
		dailyReport:       dailyReport,
		lowStock:          lowStock,
	}
}

type orderItemResponse struct {
	ProductID uint64 `json:"product_id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	UnitPrice int64  `json:"unit_price"`
	Quantity  int    `json:"quantity"`
	Subtotal  int64  `json:"subtotal"`
}

type orderResponse struct {
	ID            uint64              `json:"id"`
	UserID        uint64              `json:"user_id"`
	Status        string              `json:"status"`
	Items         []orderItemResponse `json:"items"`
	Total         int64               `json:"total"`
	Currency      string              `json:"currency"`
	FailureReason string              `json:"failure_reason,omitempty"`
	CreatedAt     string              `json:"created_at"`
}

type dailyReportResponse struct {
	Date         string         `json:"date"`
	TotalOrders  int            `json:"total_orders"`
	TotalRevenue int64          `json:"total_revenue"`
	Currency     string         `json:"currency"`
	ByStatus     map[string]int `json:"by_status"`
}

type lowStockResponse struct {
	ProductID    uint64 `json:"product_id"`
	Available    int    `json:"available"`
	Reserved     int    `json:"reserved"`
	ReorderLevel int    `json:"reorder_level"`
}

type updateOrderStatusRequest struct {
	Status string `json:"status"`
}

func (h *Admin) Register(rg *gin.RouterGroup) {
	admin := rg.Group("/admin")
	admin.GET("/orders", h.ListOrders)
	admin.PUT("/orders/:id/status", h.UpdateOrderStatus)
	admin.GET("/reports/daily", h.DailyReport)
	admin.GET("/inventory/low-stock", h.LowStock)
}

func (h *Admin) ListOrders(c *gin.Context) {
	page := parsePage(c)

	orders, total, err := h.listAllOrders.Handle(c.Request.Context(), page)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	items := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		items = append(items, toOrderResponse(o))
	}

	httpx.Page(c, items, total, page.Number, page.Size)
}

func (h *Admin) UpdateOrderStatus(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		httpx.Error(c, err)
		return
	}

	var req updateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, errs.Validation("admin.invalid_body", "request body must be valid json"))
		return
	}
	if req.Status == "" {
		httpx.Error(c, errs.Validation("admin.missing_status", "status is required"))
		return
	}

	if err := h.updateOrderStatus.Handle(c.Request.Context(), id, order.Status(req.Status)); err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, gin.H{"id": id, "status": req.Status})
}

func (h *Admin) DailyReport(c *gin.Context) {
	date := c.Query("date")

	result, err := h.dailyReport.Handle(c.Request.Context(), date)
	if err != nil {
		httpx.Error(c, err)
		return
	}

	httpx.OK(c, dailyReportResponse{
		Date:         result.Date,
		TotalOrders:  result.TotalOrders,
		TotalRevenue: result.TotalRevenue,
		Currency:     result.Currency,
		ByStatus:     result.ByStatus,
	})
}

func (h *Admin) LowStock(c *gin.Context) {
	items, err := h.lowStock.Handle(c.Request.Context())
	if err != nil {
		httpx.Error(c, err)
		return
	}

	out := make([]lowStockResponse, 0, len(items))
	for _, inv := range items {
		out = append(out, toLowStockResponse(inv))
	}

	httpx.OK(c, gin.H{"items": out})
}

func toOrderResponse(o *order.Order) orderResponse {
	items := make([]orderItemResponse, 0, len(o.Items()))
	for _, item := range o.Items() {
		items = append(items, orderItemResponse{
			ProductID: item.ProductID(),
			SKU:       item.SKU(),
			Name:      item.Name(),
			UnitPrice: item.UnitPrice().Amount(),
			Quantity:  item.Quantity(),
			Subtotal:  item.Subtotal().Amount(),
		})
	}

	return orderResponse{
		ID:            o.ID(),
		UserID:        o.UserID(),
		Status:        string(o.Status()),
		Items:         items,
		Total:         o.Total().Amount(),
		Currency:      o.Total().Currency(),
		FailureReason: o.FailureReason(),
		CreatedAt:     o.CreatedAt().Format(time.RFC3339),
	}
}

func toLowStockResponse(inv *inventory.Inventory) lowStockResponse {
	return lowStockResponse{
		ProductID:    inv.ProductID(),
		Available:    inv.Available(),
		Reserved:     inv.Reserved(),
		ReorderLevel: inv.ReorderLevel(),
	}
}

func parsePage(c *gin.Context) ports.Page {
	number := defaultPageNumber
	if raw := c.Query("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			number = parsed
		}
	}

	size := defaultPageSize
	if raw := c.Query("size"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			size = parsed
		}
	}
	if size > maxPageSize {
		size = maxPageSize
	}

	return ports.Page{Number: number, Size: size}
}

func parseUintParam(raw string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, errs.Validation("admin.invalid_id", "invalid order id")
	}
	return id, nil
}
