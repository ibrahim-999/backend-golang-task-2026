package ports

import "context"

type RepositoryProvider interface {
	Users() UserRepository
	Products() ProductRepository
	Inventory() InventoryRepository
	Orders() OrderRepository
	Payments() PaymentRepository
	Notifications() NotificationRepository
	Audit() AuditRepository
}

type UnitOfWork interface {
	Do(ctx context.Context, fn func(RepositoryProvider) error) error
}
