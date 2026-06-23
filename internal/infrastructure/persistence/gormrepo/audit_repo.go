package gormrepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/audit"
)

type auditRepo struct {
	db *gorm.DB
}

func newAuditRepo(db *gorm.DB) *auditRepo {
	return &auditRepo{db: db}
}

func (r *auditRepo) Append(ctx context.Context, e *audit.Entry) error {
	m := auditToModel(e)
	return r.db.WithContext(ctx).Create(&m).Error
}

func auditToModel(e *audit.Entry) AuditLogModel {
	return AuditLogModel{
		ID:        e.ID(),
		ActorID:   e.ActorID(),
		Action:    e.Action(),
		Entity:    e.Entity(),
		EntityID:  e.EntityID(),
		Before:    e.Before(),
		After:     e.After(),
		Metadata:  e.Metadata(),
		CreatedAt: e.CreatedAt(),
	}
}

var _ ports.AuditRepository = (*auditRepo)(nil)
