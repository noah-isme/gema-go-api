package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// AdminSubmissionRepository provides persistence helpers for grading workflows.
type AdminSubmissionRepository interface {
	GetByID(ctx context.Context, id uint) (models.Submission, error)
	Update(ctx context.Context, submission *models.Submission) error
	CreateHistory(ctx context.Context, history *models.SubmissionGradeHistory) error
}

type adminSubmissionRepository struct {
	db *gorm.DB
}

// NewAdminSubmissionRepository builds a grading-aware submission repository.
func NewAdminSubmissionRepository(db *gorm.DB) AdminSubmissionRepository {
	return &adminSubmissionRepository{db: db}
}

func (r *adminSubmissionRepository) GetByID(ctx context.Context, id uint) (models.Submission, error) {
	var submission models.Submission
	if err := r.db.WithContext(ctx).
		Preload("Assignment").
		Preload("Student").
		Preload("History", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("graded_at DESC")
		}).
		First(&submission, id).Error; err != nil {
		return models.Submission{}, err
	}

	return submission, nil
}

func (r *adminSubmissionRepository) Update(ctx context.Context, submission *models.Submission) error {
	return r.db.WithContext(ctx).Save(submission).Error
}

func (r *adminSubmissionRepository) CreateHistory(ctx context.Context, history *models.SubmissionGradeHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}
