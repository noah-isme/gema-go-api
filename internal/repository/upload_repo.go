package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// UploadRepository persists metadata about uploaded files.
type UploadRepository interface {
	Create(ctx context.Context, record *models.UploadRecord) error
}

type uploadRepository struct {
	db *gorm.DB
}

// NewUploadRepository constructs a repository for upload records.
func NewUploadRepository(db *gorm.DB) UploadRepository {
	return &uploadRepository{db: db}
}

func (r *uploadRepository) Create(ctx context.Context, record *models.UploadRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}
