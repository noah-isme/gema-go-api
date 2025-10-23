package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// ActivityLogFilter narrows activity log queries.
type ActivityLogFilter struct {
	Page       int
	PageSize   int
	ActorID    *uint
	Action     string
	EntityType string
}

// ActivityLogRepository persists audit trail events.
type ActivityLogRepository interface {
	Create(ctx context.Context, entry *models.ActivityLog) error
	List(ctx context.Context, filter ActivityLogFilter) ([]models.ActivityLog, int64, error)
	ListRecent(ctx context.Context, filter ActivityLogRecentFilter) ([]models.ActivityLog, int64, error)
}

type activityLogRepository struct {
	db *gorm.DB
}

// NewActivityLogRepository constructs the activity log repository.
func NewActivityLogRepository(db *gorm.DB) ActivityLogRepository {
	return &activityLogRepository{db: db}
}

func (r *activityLogRepository) Create(ctx context.Context, entry *models.ActivityLog) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *activityLogRepository) List(ctx context.Context, filter ActivityLogFilter) ([]models.ActivityLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ActivityLog{})

	if filter.ActorID != nil {
		query = query.Where("actor_id = ?", *filter.ActorID)
	}

	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}

	if filter.EntityType != "" {
		query = query.Where("entity_type = ?", filter.EntityType)
	}

	countQuery := query.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.PageSize > 0 {
		page := filter.Page
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	var entries []models.ActivityLog
	if err := query.Order("created_at DESC").Find(&entries).Error; err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// ActivityLogRecentFilter narrows queries for recent activity fetches.
type ActivityLogRecentFilter struct {
	Since    time.Time
	Until    time.Time
	ActorID  *uint
	Action   string
	Entity   string
	Page     int
	PageSize int
}

func (r *activityLogRepository) ListRecent(ctx context.Context, filter ActivityLogRecentFilter) ([]models.ActivityLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ActivityLog{})

	if !filter.Since.IsZero() {
		query = query.Where("created_at >= ?", filter.Since)
	}
	if !filter.Until.IsZero() {
		query = query.Where("created_at <= ?", filter.Until)
	}
	if filter.ActorID != nil {
		query = query.Where("actor_id = ?", *filter.ActorID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Entity != "" {
		query = query.Where("entity_type = ?", filter.Entity)
	}

	countQuery := query.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.PageSize > 0 {
		page := filter.Page
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	var entries []models.ActivityLog
	if err := query.Order("created_at DESC").Find(&entries).Error; err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}
