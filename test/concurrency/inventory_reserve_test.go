//go:build integration

package concurrency_test

import (
	"context"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/persistence/gormrepo"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
)

func TestReserveDoesNotOversell(t *testing.T) {
	port, err := strconv.Atoi(envOr("DB_PORT", "5432"))
	require.NoError(t, err)

	cfg := config.DB{
		Host:           envOr("DB_HOST", "localhost"),
		Port:           port,
		User:           envOr("DB_USER", "postgres"),
		Password:       envOr("DB_PASSWORD", ""),
		Name:           envOr("DB_NAME", "orders"),
		SSLMode:        envOr("DB_SSLMODE", "disable"),
		ConnectRetries: 15,
		ConnectBackoff: time.Second,
	}

	db, err := gormrepo.Open(cfg, zerolog.Nop())
	require.NoError(t, err)
	t.Cleanup(func() { db.Exec("DELETE FROM inventories WHERE product_id = ?", 1) })

	require.NoError(t, gormrepo.AutoMigrate(db))
	require.NoError(t, db.Exec("DELETE FROM inventories WHERE product_id = ?", 1).Error)
	require.NoError(t, db.Create(&gormrepo.InventoryModel{
		ProductID:    1,
		Available:    1,
		Reserved:     0,
		ReorderLevel: 0,
	}).Error)

	repo := gormrepo.NewRepositories(db).Inventory()

	const n = 500
	var successes int64
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ok, rerr := repo.Reserve(context.Background(), 1, 1)
			if rerr == nil && ok {
				atomic.AddInt64(&successes, 1)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, int64(1), atomic.LoadInt64(&successes))

	var row gormrepo.InventoryModel
	require.NoError(t, db.Where("product_id = ?", 1).First(&row).Error)
	require.Equal(t, 0, row.Available)
	require.Equal(t, 1, row.Reserved)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
