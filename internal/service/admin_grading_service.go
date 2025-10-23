package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// ErrAdminSubmissionNotFound indicates the submission was not located.
var ErrAdminSubmissionNotFound = errors.New("submission not found")

// ErrScoreExceedsMax indicates a grading score surpasses the assignment max.
var ErrScoreExceedsMax = errors.New("score exceeds assignment max")

// AdminGradingService encapsulates grading workflows for administrators and teachers.
type AdminGradingService interface {
	Grade(ctx context.Context, submissionID uint, payload dto.AdminGradeSubmissionRequest, actor ActivityActor) (dto.SubmissionResponse, error)
}

type adminGradingService struct {
	repo      repository.AdminSubmissionRepository
	validator *validator.Validate
	activity  ActivityRecorder
	logger    zerolog.Logger
	now       func() time.Time
}

// NewAdminGradingService constructs the grading service.
func NewAdminGradingService(repo repository.AdminSubmissionRepository, validator *validator.Validate, activity ActivityRecorder, logger zerolog.Logger) AdminGradingService {
	return &adminGradingService{
		repo:      repo,
		validator: validator,
		activity:  activity,
		logger:    logger.With().Str("component", "admin_grading_service").Logger(),
		now:       time.Now,
	}
}

func (s *adminGradingService) Grade(ctx context.Context, submissionID uint, payload dto.AdminGradeSubmissionRequest, actor ActivityActor) (dto.SubmissionResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.SubmissionResponse{}, err
	}

	submission, err := s.repo.GetByID(ctx, submissionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.SubmissionResponse{}, ErrAdminSubmissionNotFound
		}
		return dto.SubmissionResponse{}, err
	}

	maxScore := submission.Assignment.MaxScore
	if maxScore <= 0 {
		maxScore = 100
	}

	if payload.Score > maxScore+1e-9 {
		return dto.SubmissionResponse{}, ErrScoreExceedsMax
	}

	payloadFeedback := strings.TrimSpace(payload.Feedback)
	currentFeedback := strings.TrimSpace(submission.Feedback)
	currentScore := submission.Grade

	isIdempotent := currentScore != nil && math.Abs(*currentScore-payload.Score) < 1e-6 && currentFeedback == payloadFeedback
	if isIdempotent {
		if submission.GradedBy != nil && *submission.GradedBy == actor.ID {
			return dto.NewSubmissionResponse(submission), nil
		}
	}

	grade := payload.Score
	submission.Grade = &grade
	submission.Feedback = payloadFeedback
	submission.Status = models.SubmissionStatusGraded
	gradedAt := s.now()
	submission.GradedAt = &gradedAt
	gradedBy := actor.ID
	submission.GradedBy = &gradedBy

	if err := s.repo.Update(ctx, &submission); err != nil {
		return dto.SubmissionResponse{}, err
	}

	history := models.SubmissionGradeHistory{
		SubmissionID: submission.ID,
		Score:        payload.Score,
		Feedback:     payloadFeedback,
		GradedBy:     actor.ID,
		GradedAt:     gradedAt,
	}
	if err := s.repo.CreateHistory(ctx, &history); err != nil {
		s.logger.Warn().Err(err).Uint("submission_id", submission.ID).Msg("failed to persist grading history")
	}

	if s.activity != nil {
		metadata := map[string]interface{}{
			"submission_id": submission.ID,
			"student_id":    submission.StudentID,
			"score":         payload.Score,
			"assignment_id": submission.AssignmentID,
		}
		_, _ = s.activity.Record(ctx, ActivityEntry{
			ActorID:    actor.ID,
			ActorRole:  actor.Role,
			Action:     "submission.graded",
			EntityType: "submission",
			EntityID:   &submission.ID,
			Metadata:   metadata,
		})
	}

	return dto.NewSubmissionResponse(submission), nil
}
