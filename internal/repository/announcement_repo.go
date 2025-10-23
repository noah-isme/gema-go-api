package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// AnnouncementFilter filters announcement list queries.
type AnnouncementFilter struct {
	Page     int
	PageSize int
}

// AnnouncementRepository exposes persistence helpers for announcements.
type AnnouncementRepository interface {
	ListActive(ctx context.Context, filter AnnouncementFilter) ([]models.Announcement, int64, error)
	UpsertBatch(ctx context.Context, items []models.Announcement) (int64, error)
}

type announcementRepository struct {
	db *gorm.DB
}

// NewAnnouncementRepository constructs the repository implementation.
func NewAnnouncementRepository(db *gorm.DB) AnnouncementRepository {
	return &announcementRepository{db: db}
}

func (r *announcementRepository) ListActive(ctx context.Context, filter AnnouncementFilter) ([]models.Announcement, int64, error) {
	now := time.Now()
	query := r.db.WithContext(ctx).Model(&models.Announcement{})
	query = query.Where("is_pinned = ? OR (starts_at <= ? AND (ends_at IS NULL OR ends_at >= ?))", true, now, now)

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

	var items []models.Announcement
	if err := query.Order("is_pinned DESC, starts_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *announcementRepository) UpsertBatch(ctx context.Context, items []models.Announcement) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}

	tx := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "slug"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "body", "starts_at", "ends_at", "is_pinned", "updated_at"}),
	})

	result := tx.Create(&items)
	return result.RowsAffected, result.Error
}
