package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// SubmissionFilter allows narrowing submission queries.
type SubmissionFilter struct {
	AssignmentID *uint
	StudentID    *uint
	Status       *string
}

// SubmissionRepository defines data operations for submissions.
type SubmissionRepository interface {
	List(ctx context.Context, filter SubmissionFilter) ([]models.Submission, error)
	GetByID(ctx context.Context, id uint) (models.Submission, error)
	GetByAssignmentAndStudent(ctx context.Context, assignmentID, studentID uint) (models.Submission, error)
	Create(ctx context.Context, submission *models.Submission) error
	Update(ctx context.Context, submission *models.Submission) error
}

type submissionRepository struct {
	db *gorm.DB
}

// NewSubmissionRepository instantiates the repository.
func NewSubmissionRepository(db *gorm.DB) SubmissionRepository {
	return &submissionRepository{db: db}
}

func (r *submissionRepository) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&models.Submission{}).
		Preload("Assignment").
		Preload("Student")
}

func (r *submissionRepository) List(ctx context.Context, filter SubmissionFilter) ([]models.Submission, error) {
	query := r.baseQuery(ctx)

	if filter.AssignmentID != nil {
		query = query.Where("assignment_id = ?", *filter.AssignmentID)
	}

	if filter.StudentID != nil {
		query = query.Where("student_id = ?", *filter.StudentID)
	}

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	var submissions []models.Submission
	if err := query.Order("created_at DESC").Find(&submissions).Error; err != nil {
		return nil, err
	}

	return submissions, nil
}

func (r *submissionRepository) GetByID(ctx context.Context, id uint) (models.Submission, error) {
	var submission models.Submission
	if err := r.baseQuery(ctx).First(&submission, id).Error; err != nil {
		return models.Submission{}, err
	}

	return submission, nil
}

func (r *submissionRepository) GetByAssignmentAndStudent(ctx context.Context, assignmentID, studentID uint) (models.Submission, error) {
	var submission models.Submission
	if err := r.baseQuery(ctx).
		Where("assignment_id = ?", assignmentID).
		Where("student_id = ?", studentID).
		Order("created_at DESC").
		First(&submission).Error; err != nil {
		return models.Submission{}, err
	}

	return submission, nil
}

func (r *submissionRepository) Create(ctx context.Context, submission *models.Submission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

func (r *submissionRepository) Update(ctx context.Context, submission *models.Submission) error {
	return r.db.WithContext(ctx).Save(submission).Error
}
