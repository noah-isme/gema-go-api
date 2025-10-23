package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// AdminAnalyticsRepository supplies data for administrator analytics dashboards.
type AdminAnalyticsRepository interface {
	CountActiveStudents(ctx context.Context) (int64, error)
	ListSubmissionsWithAssignments(ctx context.Context) ([]models.Submission, error)
	ListSubmissionsSince(ctx context.Context, since time.Time) ([]models.Submission, error)
}

type adminAnalyticsRepository struct {
	db *gorm.DB
}

// NewAdminAnalyticsRepository constructs the analytics repository.
func NewAdminAnalyticsRepository(db *gorm.DB) AdminAnalyticsRepository {
	return &adminAnalyticsRepository{db: db}
}

func (r *adminAnalyticsRepository) CountActiveStudents(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Student{}).
		Where("status = ?", models.StudentStatusActive).
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}

func (r *adminAnalyticsRepository) ListSubmissionsWithAssignments(ctx context.Context) ([]models.Submission, error) {
	var submissions []models.Submission
	err := r.db.WithContext(ctx).
		Preload("Assignment").
		Preload("Student").
		Find(&submissions).Error
	return submissions, err
}

func (r *adminAnalyticsRepository) ListSubmissionsSince(ctx context.Context, since time.Time) ([]models.Submission, error) {
	var submissions []models.Submission
	err := r.db.WithContext(ctx).
		Where("created_at >= ?", since).
		Preload("Assignment").
		Preload("Student").
		Find(&submissions).Error
	return submissions, err
}
