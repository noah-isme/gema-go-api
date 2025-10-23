package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// NotificationRepository handles persistence for notification entities.
type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]models.Notification, error)
	MarkRead(ctx context.Context, id uint, userID string) (models.Notification, error)
	FindByID(ctx context.Context, id uint) (models.Notification, error)
}

type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository constructs a repository backed by GORM.
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	return r.db.WithContext(ctx).Create(notification).Error
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]models.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var notifications []models.Notification
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&notifications).Error; err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, id uint, userID string) (models.Notification, error) {
	var notification models.Notification
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&notification).Error; err != nil {
		return models.Notification{}, err
	}

	if notification.Read {
		return notification, nil
	}

	notification.Read = true
	if err := r.db.WithContext(ctx).Save(&notification).Error; err != nil {
		return models.Notification{}, err
	}

	return notification, nil
}

func (r *notificationRepository) FindByID(ctx context.Context, id uint) (models.Notification, error) {
	var notification models.Notification
	if err := r.db.WithContext(ctx).First(&notification, id).Error; err != nil {
		return models.Notification{}, err
	}
	return notification, nil
}
