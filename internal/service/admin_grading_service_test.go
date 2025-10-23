package service

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
)

type fakeAdminSubmissionRepo struct {
	submission   models.Submission
	updateCalls  int
	historyCalls int
}

func (f *fakeAdminSubmissionRepo) GetByID(ctx context.Context, id uint) (models.Submission, error) {
	return f.submission, nil
}

func (f *fakeAdminSubmissionRepo) Update(ctx context.Context, submission *models.Submission) error {
	f.updateCalls++
	f.submission = *submission
	return nil
}

func (f *fakeAdminSubmissionRepo) CreateHistory(ctx context.Context, history *models.SubmissionGradeHistory) error {
	f.historyCalls++
	return nil
}

func TestAdminGradingServiceScoreExceedsMax(t *testing.T) {
	repo := &fakeAdminSubmissionRepo{
		submission: models.Submission{
			ID:           1,
			AssignmentID: 2,
			StudentID:    3,
			Assignment: models.Assignment{
				ID:       2,
				Title:    "Essay",
				MaxScore: 50,
			},
		},
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewAdminGradingService(repo, validate, nil, testLogger())

	_, err := svc.Grade(context.Background(), 1, dto.AdminGradeSubmissionRequest{Score: 80, Feedback: "great"}, ActivityActor{ID: 10, Role: "teacher"})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrScoreExceedsMax)
	require.Equal(t, 0, repo.updateCalls)
	require.Equal(t, 0, repo.historyCalls)
}

func TestAdminGradingServiceIdempotent(t *testing.T) {
	grade := 90.0
	gradedBy := uint(42)
	gradedAt := time.Now().Add(-time.Hour)
	repo := &fakeAdminSubmissionRepo{
		submission: models.Submission{
			ID:           10,
			AssignmentID: 11,
			StudentID:    12,
			Grade:        &grade,
			Feedback:     "Well done",
			GradedBy:     &gradedBy,
			GradedAt:     &gradedAt,
			Assignment: models.Assignment{
				ID:       11,
				Title:    "Project",
				MaxScore: 100,
			},
		},
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewAdminGradingService(repo, validate, nil, testLogger())

	result, err := svc.Grade(context.Background(), 10, dto.AdminGradeSubmissionRequest{Score: 90, Feedback: "Well done"}, ActivityActor{ID: gradedBy, Role: "teacher"})
	require.NoError(t, err)
	require.Equal(t, grade, *result.Grade)
	require.Equal(t, 0, repo.updateCalls)
	require.Equal(t, 0, repo.historyCalls)
}
