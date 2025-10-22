package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// AssignmentRepository defines persistence operations for assignments.
type AssignmentRepository interface {
	List(ctx context.Context) ([]models.Assignment, error)
	GetByID(ctx context.Context, id uint) (models.Assignment, error)
	Create(ctx context.Context, assignment *models.Assignment) error
	Update(ctx context.Context, assignment *models.Assignment) error
	Delete(ctx context.Context, id uint) error
}

type assignmentRepository struct {
	db *gorm.DB
}

// NewAssignmentRepository instantiates a GORM-backed repository.
func NewAssignmentRepository(db *gorm.DB) AssignmentRepository {
	return &assignmentRepository{db: db}
}

func (r *assignmentRepository) List(ctx context.Context) ([]models.Assignment, error) {
	var assignments []models.Assignment
	if err := r.db.WithContext(ctx).Order("due_date ASC").Find(&assignments).Error; err != nil {
		return nil, err
	}

	return assignments, nil
}

func (r *assignmentRepository) GetByID(ctx context.Context, id uint) (models.Assignment, error) {
	var assignment models.Assignment
	if err := r.db.WithContext(ctx).First(&assignment, id).Error; err != nil {
		return models.Assignment{}, err
	}

	return assignment, nil
}

func (r *assignmentRepository) Create(ctx context.Context, assignment *models.Assignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

func (r *assignmentRepository) Update(ctx context.Context, assignment *models.Assignment) error {
	return r.db.WithContext(ctx).Save(assignment).Error
}

func (r *assignmentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Assignment{}, id).Error
}
