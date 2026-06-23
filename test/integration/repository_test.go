//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/audit"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/notification"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/payment"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/persistence/gormrepo"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestRepositories(t *testing.T) {
	port, err := strconv.Atoi(getenv("DB_PORT", "5432"))
	require.NoError(t, err)

	cfg := config.DB{
		Host:           getenv("DB_HOST", "localhost"),
		Port:           port,
		User:           getenv("DB_USER", "postgres"),
		Password:       getenv("DB_PASSWORD", ""),
		Name:           getenv("DB_NAME", "orders"),
		SSLMode:        getenv("DB_SSLMODE", "disable"),
		ConnectRetries: 15,
		ConnectBackoff: time.Second,
	}

	db, err := gormrepo.Open(cfg, zerolog.Nop())
	require.NoError(t, err)
	require.NoError(t, gormrepo.AutoMigrate(db))
	require.NoError(t, db.Exec("TRUNCATE users, products, inventories, orders, order_items, payments, notifications, audit_logs RESTART IDENTITY CASCADE").Error)

	ctx := context.Background()
	repos := gormrepo.NewRepositories(db)
	uow := gormrepo.NewUnitOfWork(db)

	u, err := user.NewUser("repo-user@example.com", "hashed-pw", "Repo User", user.RoleCustomer)
	require.NoError(t, err)
	require.NoError(t, repos.Users().Create(ctx, u))
	require.NotZero(t, u.ID())

	fetchedUser, err := repos.Users().FindByID(ctx, u.ID())
	require.NoError(t, err)
	assert.Equal(t, "repo-user@example.com", fetchedUser.Email())

	byEmail, err := repos.Users().FindByEmail(ctx, "repo-user@example.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID(), byEmail.ID())

	u.UpdateProfile("Repo User Renamed")
	require.NoError(t, repos.Users().Update(ctx, u))

	_, err = repos.Users().FindByID(ctx, 9999999)
	assert.Equal(t, errs.KindNotFound, errs.KindOf(err))
	_, err = repos.Users().FindByEmail(ctx, "missing@example.com")
	assert.Equal(t, errs.KindNotFound, errs.KindOf(err))

	p, err := product.NewProduct("REPO-SKU-1", "Repo Product", "a product", shared.MustMoney(1500, "USD"))
	require.NoError(t, err)
	require.NoError(t, repos.Products().Create(ctx, p))
	require.NotZero(t, p.ID())

	fetchedProduct, err := repos.Products().FindByID(ctx, p.ID())
	require.NoError(t, err)
	assert.Equal(t, "REPO-SKU-1", fetchedProduct.SKU())

	products, total, err := repos.Products().List(ctx, ports.Page{Number: 1, Size: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	assert.NotEmpty(t, products)

	p.Reprice(shared.MustMoney(1800, "USD"))
	require.NoError(t, repos.Products().Update(ctx, p))
	_, err = repos.Products().FindByID(ctx, 9999999)
	assert.Equal(t, errs.KindNotFound, errs.KindOf(err))

	inv, err := inventory.NewInventory(p.ID(), 100, 10)
	require.NoError(t, err)
	require.NoError(t, repos.Inventory().Save(ctx, inv))

	fetchedInv, err := repos.Inventory().FindByProductID(ctx, p.ID())
	require.NoError(t, err)
	assert.Equal(t, 100, fetchedInv.Available())

	ok, err := repos.Inventory().Reserve(ctx, p.ID(), 5)
	require.NoError(t, err)
	assert.True(t, ok)
	require.NoError(t, repos.Inventory().Commit(ctx, p.ID(), 2))
	require.NoError(t, repos.Inventory().Release(ctx, p.ID(), 3))
	require.NoError(t, repos.Inventory().Restock(ctx, p.ID(), 50))

	lowInv, err := inventory.NewInventory(p.ID()+1000, 1, 5)
	require.NoError(t, err)
	require.NoError(t, repos.Inventory().Save(ctx, lowInv))
	low, err := repos.Inventory().ListLowStock(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, low)

	item, err := order.NewItem(p.ID(), p.SKU(), p.Name(), p.Price(), 2)
	require.NoError(t, err)
	ord, err := order.NewOrder(u.ID(), []order.Item{item}, "repo-idem-1")
	require.NoError(t, err)
	require.NoError(t, uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		return rp.Orders().Create(ctx, ord)
	}))
	require.NotZero(t, ord.ID())

	fetchedOrder, err := repos.Orders().FindByID(ctx, ord.ID())
	require.NoError(t, err)
	assert.Len(t, fetchedOrder.Items(), 1)

	byKey, err := repos.Orders().FindByIdempotencyKey(ctx, "repo-idem-1")
	require.NoError(t, err)
	assert.Equal(t, ord.ID(), byKey.ID())

	require.NoError(t, ord.MarkReserved())
	require.NoError(t, repos.Orders().Update(ctx, ord))

	byUser, userTotal, err := repos.Orders().ListByUser(ctx, u.ID(), ports.Page{Number: 1, Size: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, userTotal, int64(1))
	assert.NotEmpty(t, byUser)

	allOrders, allTotal, err := repos.Orders().ListAll(ctx, ports.Page{Number: 1, Size: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, allTotal, int64(1))
	assert.NotEmpty(t, allOrders)

	_, err = repos.Orders().FindByID(ctx, 9999999)
	assert.Equal(t, errs.KindNotFound, errs.KindOf(err))

	pay, err := payment.NewPayment(ord.ID(), "repo-idem-1", ord.Total())
	require.NoError(t, err)
	require.NoError(t, repos.Payments().Create(ctx, pay))
	require.NotZero(t, pay.ID())

	_, err = repos.Payments().FindByID(ctx, pay.ID())
	require.NoError(t, err)
	byOrder, err := repos.Payments().FindByOrderID(ctx, ord.ID())
	require.NoError(t, err)
	assert.Equal(t, pay.ID(), byOrder.ID())
	_, err = repos.Payments().FindByIdempotencyKey(ctx, "repo-idem-1")
	require.NoError(t, err)

	require.NoError(t, pay.MarkSucceeded("provider-ref-1"))
	require.NoError(t, repos.Payments().Update(ctx, pay))
	_, err = repos.Payments().FindByOrderID(ctx, 9999999)
	assert.Equal(t, errs.KindNotFound, errs.KindOf(err))

	uid := u.ID()
	notif, err := notification.NewNotification(&uid, notification.TypeOrderConfirmed, notification.ChannelEmail, "Order confirmed", "{}")
	require.NoError(t, err)
	require.NoError(t, repos.Notifications().Create(ctx, notif))

	queued, err := repos.Notifications().ListByStatus(ctx, notification.StatusQueued, 10)
	require.NoError(t, err)
	assert.NotEmpty(t, queued)

	require.NoError(t, notif.MarkSent())
	require.NoError(t, repos.Notifications().Update(ctx, notif))

	entry, err := audit.NewEntry(&uid, "order.created", "order", ord.ID(), "", "{}", "seed")
	require.NoError(t, err)
	require.NoError(t, repos.Audit().Append(ctx, entry))

	rollbackErr := uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		rolled, _ := user.NewUser("rollback@example.com", "h", "Rolled Back", user.RoleCustomer)
		if err := rp.Users().Create(ctx, rolled); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	require.Error(t, rollbackErr)
	_, err = repos.Users().FindByEmail(ctx, "rollback@example.com")
	assert.Equal(t, errs.KindNotFound, errs.KindOf(err), "transaction should have rolled back")
}
