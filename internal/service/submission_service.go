package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// ErrSubmissionNotFound indicates a submission could not be found.
var ErrSubmissionNotFound = errors.New("submission not found")

// SubmissionService orchestrates submission workflows.
type SubmissionService interface {
	List(ctx context.Context, filter dto.SubmissionFilter) ([]dto.SubmissionResponse, error)
	Create(ctx context.Context, payload dto.SubmissionCreateRequest, file *multipart.FileHeader) (dto.SubmissionResponse, error)
	Update(ctx context.Context, id uint, payload dto.SubmissionUpdateRequest) (dto.SubmissionResponse, error)
}

type submissionService struct {
	submissions repository.SubmissionRepository
	assignments repository.AssignmentRepository
	validator   *validator.Validate
	uploader    FileUploader
	logger      zerolog.Logger
	now         func() time.Time
}

// NewSubmissionService constructs a SubmissionService instance.
func NewSubmissionService(subRepo repository.SubmissionRepository, assignmentRepo repository.AssignmentRepository, validate *validator.Validate, uploader FileUploader, logger zerolog.Logger) SubmissionService {
	return &submissionService{
		submissions: subRepo,
		assignments: assignmentRepo,
		validator:   validate,
		uploader:    uploader,
		logger:      logger.With().Str("component", "submission_service").Logger(),
		now:         time.Now,
	}
}

func (s *submissionService) List(ctx context.Context, filter dto.SubmissionFilter) ([]dto.SubmissionResponse, error) {
	if err := s.validator.Struct(filter); err != nil {
		return nil, err
	}

	repoFilter := repository.SubmissionFilter{
		AssignmentID: filter.AssignmentID,
		StudentID:    filter.StudentID,
		Status:       filter.Status,
	}

	submissions, err := s.submissions.List(ctx, repoFilter)
	if err != nil {
		return nil, err
	}

	return dto.NewSubmissionResponseSlice(submissions), nil
}

func (s *submissionService) Create(ctx context.Context, payload dto.SubmissionCreateRequest, file *multipart.FileHeader) (dto.SubmissionResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.SubmissionResponse{}, err
	}

	if file == nil {
		return dto.SubmissionResponse{}, fmt.Errorf("submission file is required")
	}

	assignment, err := s.assignments.GetByID(ctx, payload.AssignmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.SubmissionResponse{}, ErrAssignmentNotFound
		}
		return dto.SubmissionResponse{}, err
	}

	if assignment.IsPastDue(s.now()) {
		return dto.SubmissionResponse{}, fmt.Errorf("assignment is past due")
	}

	if err := validateFileType(file); err != nil {
		return dto.SubmissionResponse{}, err
	}

	reader, err := file.Open()
	if err != nil {
		return dto.SubmissionResponse{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	uploadURL, err := s.uploader.Upload(ctx, file.Filename, reader)
	if err != nil {
		return dto.SubmissionResponse{}, fmt.Errorf("failed to upload file: %w", err)
	}

	submission := models.Submission{
		AssignmentID: payload.AssignmentID,
		StudentID:    payload.StudentID,
		FileURL:      uploadURL,
		Status:       models.SubmissionStatusSubmitted,
	}

	if err := s.submissions.Create(ctx, &submission); err != nil {
		return dto.SubmissionResponse{}, err
	}

	// Reload with associations
	created, err := s.submissions.GetByID(ctx, submission.ID)
	if err != nil {
		return dto.SubmissionResponse{}, err
	}

	s.logger.Info().Uint("submission_id", created.ID).Msg("submission created")

	return dto.NewSubmissionResponse(created), nil
}

func (s *submissionService) Update(ctx context.Context, id uint, payload dto.SubmissionUpdateRequest) (dto.SubmissionResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.SubmissionResponse{}, err
	}

	submission, err := s.submissions.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.SubmissionResponse{}, ErrSubmissionNotFound
		}
		return dto.SubmissionResponse{}, err
	}

	if payload.Status != nil {
		status := strings.ToLower(*payload.Status)
		if status == models.SubmissionStatusGraded && payload.Grade == nil {
			return dto.SubmissionResponse{}, fmt.Errorf("grade is required when marking as graded")
		}
		submission.Status = status
	}

	if payload.Grade != nil {
		submission.Grade = payload.Grade
		if submission.Status != models.SubmissionStatusGraded {
			submission.Status = models.SubmissionStatusGraded
		}
	}

	if payload.Feedback != nil {
		submission.Feedback = *payload.Feedback
	}

	if err := s.submissions.Update(ctx, &submission); err != nil {
		return dto.SubmissionResponse{}, err
	}

	updated, err := s.submissions.GetByID(ctx, submission.ID)
	if err != nil {
		return dto.SubmissionResponse{}, err
	}

	s.logger.Info().Uint("submission_id", submission.ID).Msg("submission updated")

	return dto.NewSubmissionResponse(updated), nil
}

func validateFileType(file *multipart.FileHeader) error {
	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	mime, err := mimetype.DetectReader(reader)
	if err != nil {
		return fmt.Errorf("failed to detect file type: %w", err)
	}

	allowed := []string{"application/pdf", "application/zip", "application/x-zip-compressed", "text/plain"}
	for _, a := range allowed {
		if mime.Is(a) {
			return nil
		}
	}

	return fmt.Errorf("unsupported file type: %s", mime.String())
}
