package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// CodingSubmissionRepository exposes persistence helpers for coding submissions.
type CodingSubmissionRepository interface {
	Create(ctx context.Context, submission *models.CodingSubmission) error
	Update(ctx context.Context, submission *models.CodingSubmission) error
	GetByID(ctx context.Context, id uint) (models.CodingSubmission, error)
	SaveEvaluation(ctx context.Context, evaluation *models.CodingEvaluation) error
}

// NewCodingSubmissionRepository constructs a coding submission repository.
func NewCodingSubmissionRepository(db *gorm.DB) CodingSubmissionRepository {
	return &codingSubmissionRepository{db: db}
}

type codingSubmissionRepository struct {
	db *gorm.DB
}

func (r *codingSubmissionRepository) Create(ctx context.Context, submission *models.CodingSubmission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

func (r *codingSubmissionRepository) Update(ctx context.Context, submission *models.CodingSubmission) error {
	return r.db.WithContext(ctx).Save(submission).Error
}

func (r *codingSubmissionRepository) GetByID(ctx context.Context, id uint) (models.CodingSubmission, error) {
	var submission models.CodingSubmission
	err := r.db.WithContext(ctx).
		Preload("Task").
		Preload("Evaluations").
		First(&submission, id).Error
	if err != nil {
		return models.CodingSubmission{}, err
	}
	return submission, nil
}

func (r *codingSubmissionRepository) SaveEvaluation(ctx context.Context, evaluation *models.CodingEvaluation) error {
	return r.db.WithContext(ctx).Create(evaluation).Error
}
