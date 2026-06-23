package gormrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/inventory"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type inventoryRepo struct {
	db *gorm.DB
}

func newInventoryRepo(db *gorm.DB) *inventoryRepo {
	return &inventoryRepo{db: db}
}

func (r *inventoryRepo) FindByProductID(ctx context.Context, productID uint64) (*inventory.Inventory, error) {
	var m InventoryModel
	if err := r.db.WithContext(ctx).Where("product_id = ?", productID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("inventory.not_found", "inventory not found for product")
		}
		return nil, err
	}
	return inventoryToDomain(m), nil
}

func (r *inventoryRepo) Save(ctx context.Context, inv *inventory.Inventory) error {
	m := inventoryToModel(inv)
	return r.db.WithContext(ctx).Save(&m).Error
}

func (r *inventoryRepo) Reserve(ctx context.Context, productID uint64, qty int) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&InventoryModel{}).
		Where("product_id = ? AND available >= ?", productID, qty).
		Updates(map[string]any{
			"available": gorm.Expr("available - ?", qty),
			"reserved":  gorm.Expr("reserved + ?", qty),
			"version":   gorm.Expr("version + 1"),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *inventoryRepo) Release(ctx context.Context, productID uint64, qty int) error {
	res := r.db.WithContext(ctx).
		Model(&InventoryModel{}).
		Where("product_id = ? AND reserved >= ?", productID, qty).
		Updates(map[string]any{
			"available": gorm.Expr("available + ?", qty),
			"reserved":  gorm.Expr("reserved - ?", qty),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.Conflict("inventory.release_failed", "cannot release more than is reserved")
	}
	return nil
}

func (r *inventoryRepo) Commit(ctx context.Context, productID uint64, qty int) error {
	res := r.db.WithContext(ctx).
		Model(&InventoryModel{}).
		Where("product_id = ? AND reserved >= ?", productID, qty).
		Update("reserved", gorm.Expr("reserved - ?", qty))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.Conflict("inventory.commit_failed", "cannot commit more than is reserved")
	}
	return nil
}

func (r *inventoryRepo) Restock(ctx context.Context, productID uint64, qty int) error {
	return r.db.WithContext(ctx).
		Model(&InventoryModel{}).
		Where("product_id = ?", productID).
		Updates(map[string]any{
			"available": gorm.Expr("available + ?", qty),
			"version":   gorm.Expr("version + 1"),
		}).Error
}

func (r *inventoryRepo) ListLowStock(ctx context.Context) ([]*inventory.Inventory, error) {
	var ms []InventoryModel
	if err := r.db.WithContext(ctx).Where("available <= reorder_level").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]*inventory.Inventory, 0, len(ms))
	for _, m := range ms {
		out = append(out, inventoryToDomain(m))
	}
	return out, nil
}

func inventoryToModel(inv *inventory.Inventory) InventoryModel {
	return InventoryModel{
		ID:           inv.ID(),
		ProductID:    inv.ProductID(),
		Available:    inv.Available(),
		Reserved:     inv.Reserved(),
		ReorderLevel: inv.ReorderLevel(),
		Version:      inv.Version(),
	}
}

func inventoryToDomain(m InventoryModel) *inventory.Inventory {
	return inventory.ReconstituteInventory(m.ID, m.ProductID, m.Available, m.Reserved, m.ReorderLevel, m.Version)
}

var _ ports.InventoryRepository = (*inventoryRepo)(nil)
