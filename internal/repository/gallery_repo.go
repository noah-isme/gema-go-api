package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// GalleryFilter narrows public gallery queries.
type GalleryFilter struct {
	Tags     []string
	Search   string
	Page     int
	PageSize int
}

// GalleryRepository manages gallery persistence operations.
type GalleryRepository interface {
	List(ctx context.Context, filter GalleryFilter) ([]models.GalleryItem, int64, error)
	UpsertBatch(ctx context.Context, items []models.GalleryItem) (int64, error)
	GetByID(ctx context.Context, id uint) (models.GalleryItem, error)
	Create(ctx context.Context, item *models.GalleryItem) error
	Update(ctx context.Context, item *models.GalleryItem) error
	Delete(ctx context.Context, id uint) error
}

type galleryRepository struct {
	db *gorm.DB
}

// NewGalleryRepository constructs a gallery repository implementation.
func NewGalleryRepository(db *gorm.DB) GalleryRepository {
	return &galleryRepository{db: db}
}

func (r *galleryRepository) List(ctx context.Context, filter GalleryFilter) ([]models.GalleryItem, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.GalleryItem{})

	for _, tag := range filter.Tags {
		trimmed := strings.TrimSpace(strings.ToLower(tag))
		if trimmed == "" {
			continue
		}
		like := "%|" + trimmed + "|%"
		query = query.Where("tags LIKE ?", like)
	}

	if filter.Search != "" {
		pattern := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(caption) LIKE ?", pattern, pattern)
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

	var items []models.GalleryItem
	if err := query.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *galleryRepository) UpsertBatch(ctx context.Context, items []models.GalleryItem) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}

	tx := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "slug"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "caption", "image_path", "tags", "updated_at"}),
	})

	result := tx.Create(&items)
	return result.RowsAffected, result.Error
}

func (r *galleryRepository) GetByID(ctx context.Context, id uint) (models.GalleryItem, error) {
	var item models.GalleryItem
	err := r.db.WithContext(ctx).First(&item, id).Error
	return item, err
}

func (r *galleryRepository) Create(ctx context.Context, item *models.GalleryItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *galleryRepository) Update(ctx context.Context, item *models.GalleryItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *galleryRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.GalleryItem{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
