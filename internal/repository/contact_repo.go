package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// ContactRepository persists contact form submissions.
type ContactRepository interface {
	Create(ctx context.Context, submission *models.ContactSubmission) error
	UpdateStatus(ctx context.Context, id uint, status string) error
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
