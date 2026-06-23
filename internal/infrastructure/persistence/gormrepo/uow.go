package gormrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
)

type repositoryProvider struct {
	db *gorm.DB
}

func NewRepositories(db *gorm.DB) ports.RepositoryProvider {
	return &repositoryProvider{db: db}
}

func (p *repositoryProvider) Users() ports.UserRepository {
	return newUserRepo(p.db)
}

func (p *repositoryProvider) Products() ports.ProductRepository {
	return newProductRepo(p.db)
}

func (p *repositoryProvider) Inventory() ports.InventoryRepository {
	return newInventoryRepo(p.db)
}

func (p *repositoryProvider) Orders() ports.OrderRepository {
	return newOrderRepo(p.db)
}

func (p *repositoryProvider) Payments() ports.PaymentRepository {
	return newPaymentRepo(p.db)
}

func (p *repositoryProvider) Notifications() ports.NotificationRepository {
	return newNotificationRepo(p.db)
}

func (p *repositoryProvider) Audit() ports.AuditRepository {
	return newAuditRepo(p.db)
}

type unitOfWork struct {
	db *gorm.DB
}

func NewUnitOfWork(db *gorm.DB) ports.UnitOfWork {
	return &unitOfWork{db: db}
}

func (u *unitOfWork) Do(ctx context.Context, fn func(ports.RepositoryProvider) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repositoryProvider{db: tx})
	})
}

var _ ports.RepositoryProvider = (*repositoryProvider)(nil)
var _ ports.UnitOfWork = (*unitOfWork)(nil)
