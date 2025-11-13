package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// AdminContactFilter narrows admin contact queries.
type AdminContactFilter struct {
	Search   string
	Status   string
	Sort     string
	Page     int
	PageSize int
}

// ContactRepository persists contact form submissions.
type ContactRepository interface {
	Create(ctx context.Context, submission *models.ContactSubmission) error
	UpdateStatus(ctx context.Context, id uint, status string) error
	List(ctx context.Context, filter AdminContactFilter) ([]models.ContactSubmission, int64, error)
	GetByID(ctx context.Context, id uint) (models.ContactSubmission, error)
}

type contactRepository struct {
	db *gorm.DB
}

// NewContactRepository constructs a repository backed by GORM.
func NewContactRepository(db *gorm.DB) ContactRepository {
	return &contactRepository{db: db}
}

func (r *contactRepository) Create(ctx context.Context, submission *models.ContactSubmission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

func (r *contactRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.ContactSubmission{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"status": status}).
		Error
}

func (r *contactRepository) List(ctx context.Context, filter AdminContactFilter) ([]models.ContactSubmission, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ContactSubmission{})

	if filter.Search != "" {
		pattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(message) LIKE ?", pattern, pattern, pattern)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", strings.ToLower(strings.TrimSpace(filter.Status)))
	}

	countQuery := query.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sort := strings.TrimSpace(filter.Sort)
	if sort == "" {
		sort = "created_at DESC"
	}
	query = query.Order(sort)

	if filter.PageSize > 0 {
		page := filter.Page
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	var submissions []models.ContactSubmission
	if err := query.Find(&submissions).Error; err != nil {
		return nil, 0, err
	}

	return submissions, total, nil
}

func (r *contactRepository) GetByID(ctx context.Context, id uint) (models.ContactSubmission, error) {
	var submission models.ContactSubmission
	err := r.db.WithContext(ctx).First(&submission, id).Error
	return submission, err
}
