package ports

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/audit"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/notification"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/payment"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
)

type Page struct {
	Number int
	Size   int
}

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	Update(ctx context.Context, u *user.User) error
	FindByID(ctx context.Context, id uint64) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
}

type ProductRepository interface {
	Create(ctx context.Context, p *product.Product) error
	Update(ctx context.Context, p *product.Product) error
	FindByID(ctx context.Context, id uint64) (*product.Product, error)
	List(ctx context.Context, page Page) ([]*product.Product, int64, error)
}

type InventoryRepository interface {
	FindByProductID(ctx context.Context, productID uint64) (*inventory.Inventory, error)
	Save(ctx context.Context, inv *inventory.Inventory) error
	Reserve(ctx context.Context, productID uint64, qty int) (bool, error)
	Release(ctx context.Context, productID uint64, qty int) error
	Commit(ctx context.Context, productID uint64, qty int) error
	Restock(ctx context.Context, productID uint64, qty int) error
	ListLowStock(ctx context.Context) ([]*inventory.Inventory, error)
}

type OrderRepository interface {
	Create(ctx context.Context, o *order.Order) error
	Update(ctx context.Context, o *order.Order) error
	FindByID(ctx context.Context, id uint64) (*order.Order, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*order.Order, error)
	ListByUser(ctx context.Context, userID uint64, page Page) ([]*order.Order, int64, error)
	ListAll(ctx context.Context, page Page) ([]*order.Order, int64, error)
}

type PaymentRepository interface {
	Create(ctx context.Context, p *payment.Payment) error
	Update(ctx context.Context, p *payment.Payment) error
	FindByID(ctx context.Context, id uint64) (*payment.Payment, error)
	FindByOrderID(ctx context.Context, orderID uint64) (*payment.Payment, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*payment.Payment, error)
}

type NotificationRepository interface {
	Create(ctx context.Context, n *notification.Notification) error
	Update(ctx context.Context, n *notification.Notification) error
	ListByStatus(ctx context.Context, status notification.Status, limit int) ([]*notification.Notification, error)
}

type AuditRepository interface {
	Append(ctx context.Context, e *audit.Entry) error
}
