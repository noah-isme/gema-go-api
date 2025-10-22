package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// WebAssignmentRepository exposes read operations for web lab assignments.
type WebAssignmentRepository interface {
	List(ctx context.Context) ([]models.WebAssignment, error)
	GetByID(ctx context.Context, id uint) (models.WebAssignment, error)
}

type webAssignmentRepository struct {
	db *gorm.DB
}

// NewWebAssignmentRepository constructs a web assignment repository.
func NewWebAssignmentRepository(db *gorm.DB) WebAssignmentRepository {
	return &webAssignmentRepository{db: db}
}

func (r *webAssignmentRepository) List(ctx context.Context) ([]models.WebAssignment, error) {
	var assignments []models.WebAssignment
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&assignments).Error; err != nil {
		return nil, err
	}

	return assignments, nil
}

func (r *webAssignmentRepository) GetByID(ctx context.Context, id uint) (models.WebAssignment, error) {
	var assignment models.WebAssignment
	if err := r.db.WithContext(ctx).First(&assignment, id).Error; err != nil {
		return models.WebAssignment{}, err
	}

	return assignment, nil
}

// WebSubmissionRepository exposes persistence helpers for web lab submissions.
type WebSubmissionRepository interface {
	Create(ctx context.Context, submission *models.WebSubmission) error
	Update(ctx context.Context, submission *models.WebSubmission) error
	GetByID(ctx context.Context, id uint) (models.WebSubmission, error)
}

type webSubmissionRepository struct {
	db *gorm.DB
}

// NewWebSubmissionRepository constructs a web submission repository.
func NewWebSubmissionRepository(db *gorm.DB) WebSubmissionRepository {
	return &webSubmissionRepository{db: db}
}

func (r *webSubmissionRepository) Create(ctx context.Context, submission *models.WebSubmission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

func (r *webSubmissionRepository) Update(ctx context.Context, submission *models.WebSubmission) error {
	return r.db.WithContext(ctx).Save(submission).Error
}

func (r *webSubmissionRepository) GetByID(ctx context.Context, id uint) (models.WebSubmission, error) {
	var submission models.WebSubmission
	if err := r.db.WithContext(ctx).
		Preload("Assignment").
		First(&submission, id).Error; err != nil {
		return models.WebSubmission{}, err
	}

	return submission, nil
}
