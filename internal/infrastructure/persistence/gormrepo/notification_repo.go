package gormrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/notification"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type notificationRepo struct {
	db *gorm.DB
}

func newNotificationRepo(db *gorm.DB) *notificationRepo {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) Create(ctx context.Context, n *notification.Notification) error {
	m := notificationToModel(n)
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *notificationRepo) Update(ctx context.Context, n *notification.Notification) error {
	m := notificationToModel(n)
	res := r.db.WithContext(ctx).Model(&NotificationModel{}).Where("id = ?", m.ID).Updates(map[string]any{
		"user_id":        m.UserID,
		"type":           m.Type,
		"channel":        m.Channel,
		"status":         m.Status,
		"subject":        m.Subject,
		"payload":        m.Payload,
		"attempts":       m.Attempts,
		"failure_reason": m.FailureReason,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("notification.not_found", "notification not found")
	}
	return nil
}

func (r *notificationRepo) ListByStatus(ctx context.Context, status notification.Status, limit int) ([]*notification.Notification, error) {
	var ms []NotificationModel
	q := r.db.WithContext(ctx).Where("status = ?", string(status))
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&ms).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("notification.not_found", "notification not found")
		}
		return nil, err
	}
	out := make([]*notification.Notification, 0, len(ms))
	for _, m := range ms {
		out = append(out, notificationToDomain(m))
	}
	return out, nil
}

func notificationToModel(n *notification.Notification) NotificationModel {
	return NotificationModel{
		ID:            n.ID(),
		UserID:        n.UserID(),
		Type:          string(n.Type()),
		Channel:       string(n.Channel()),
		Status:        string(n.Status()),
		Subject:       n.Subject(),
		Payload:       n.Payload(),
		Attempts:      n.Attempts(),
		FailureReason: n.FailureReason(),
	}
}

func notificationToDomain(m NotificationModel) *notification.Notification {
	return notification.ReconstituteNotification(
		m.ID,
		m.UserID,
		notification.Type(m.Type),
		notification.Channel(m.Channel),
		notification.Status(m.Status),
		m.Subject,
		m.Payload,
		m.Attempts,
		m.FailureReason,
		m.CreatedAt,
	)
}

var _ ports.NotificationRepository = (*notificationRepo)(nil)
