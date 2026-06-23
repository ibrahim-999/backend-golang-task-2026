package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/product"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/auth"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/persistence/gormrepo"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
)

type userSeed struct {
	email    string
	password string
	name     string
	role     user.Role
}

type productSeed struct {
	sku     string
	name    string
	desc    string
	price   int64
	stock   int
	reorder int
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := gormrepo.Open(cfg.DB, zerolog.Nop())
	if err != nil {
		return err
	}
	if err := gormrepo.AutoMigrate(db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	fmt.Printf("migrations applied: %d tables\n", len(gormrepo.AllModels()))

	ctx := context.Background()
	repos := gormrepo.NewRepositories(db)
	hasher := auth.NewBcryptHasher()

	users := []userSeed{
		{"admin@ex.com", "adminpass123", "Admin User", user.RoleAdmin},
		{"cathy@ex.com", "custpass123", "Cathy Customer", user.RoleCustomer},
		{"dave@ex.com", "custpass123", "Dave Customer", user.RoleCustomer},
		{"erin@ex.com", "custpass123", "Erin Customer", user.RoleCustomer},
	}
	createdUsers := 0
	for _, u := range users {
		made, err := seedUser(ctx, db, repos, hasher, u)
		if err != nil {
			return err
		}
		if made {
			createdUsers++
		}
	}

	products := []productSeed{
		{"SKU-1000", "Mechanical Keyboard", "Hot-swappable RGB mechanical keyboard", 8999, 50, 10},
		{"SKU-1001", "Wireless Mouse", "Ergonomic wireless mouse", 2499, 100, 20},
		{"SKU-1002", "27in 4K Monitor", "27-inch 4K IPS display", 24999, 15, 5},
		{"SKU-1003", "USB-C Hub", "7-in-1 USB-C hub", 3999, 3, 5},
		{"SKU-1004", "Laptop Stand", "Aluminium laptop stand", 4599, 0, 5},
		{"SKU-1005", "1080p Webcam", "Full-HD webcam with microphone", 5999, 8, 10},
		{"SKU-1006", "LED Desk Lamp", "Dimmable LED desk lamp", 1999, 200, 25},
		{"SKU-1007", "ANC Headphones", "Noise-cancelling over-ear headphones", 19999, 1, 3},
		{"SKU-1008", "Standing Desk", "Electric sit-stand desk", 39999, 12, 4},
		{"SKU-1009", "Ergonomic Chair", "Mesh ergonomic office chair", 28999, 7, 6},
	}
	createdProducts := 0
	for _, p := range products {
		made, err := seedProduct(ctx, db, repos, p)
		if err != nil {
			return err
		}
		if made {
			createdProducts++
		}
	}

	fmt.Printf("seed complete: %d/%d users, %d/%d products\n", createdUsers, len(users), createdProducts, len(products))
	fmt.Println("low-stock seeded: SKU-1003, SKU-1005, SKU-1007 | out-of-stock: SKU-1004")
	fmt.Println("login: admin@ex.com / adminpass123  (admin)   cathy@ex.com / custpass123  (customer)")
	return nil
}

func seedUser(ctx context.Context, db *gorm.DB, repos ports.RepositoryProvider, hasher ports.PasswordHasher, u userSeed) (bool, error) {
	var count int64
	if err := db.WithContext(ctx).Model(&gormrepo.UserModel{}).Where("email = ?", u.email).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	hash, err := hasher.Hash(u.password)
	if err != nil {
		return false, err
	}
	entity, err := user.NewUser(u.email, hash, u.name, u.role)
	if err != nil {
		return false, err
	}
	if err := repos.Users().Create(ctx, entity); err != nil {
		return false, err
	}
	return true, nil
}

func seedProduct(ctx context.Context, db *gorm.DB, repos ports.RepositoryProvider, p productSeed) (bool, error) {
	var count int64
	if err := db.WithContext(ctx).Model(&gormrepo.ProductModel{}).Where("sku = ?", p.sku).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	prod, err := product.NewProduct(p.sku, p.name, p.desc, shared.MustMoney(p.price, "USD"))
	if err != nil {
		return false, err
	}
	if err := repos.Products().Create(ctx, prod); err != nil {
		return false, err
	}
	inv, err := inventory.NewInventory(prod.ID(), p.stock, p.reorder)
	if err != nil {
		return false, err
	}
	if err := repos.Inventory().Save(ctx, inv); err != nil {
		return false, err
	}
	return true, nil
}
