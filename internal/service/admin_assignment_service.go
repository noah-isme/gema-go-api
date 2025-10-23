package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// ErrAdminAssignmentNotFound indicates the assignment was not found for admin actions.
var ErrAdminAssignmentNotFound = errors.New("admin assignment not found")

// ErrAdminAssignmentInvalidDueDate indicates the due date is invalid.
var ErrAdminAssignmentInvalidDueDate = errors.New("assignment due date must be in the future")

// AdminAssignmentService manages assignment CRUD for administrators.
type AdminAssignmentService interface {
	Create(ctx context.Context, payload dto.AdminAssignmentCreateRequest, actor ActivityActor) (dto.AdminAssignmentResponse, error)
	Update(ctx context.Context, id uint, payload dto.AdminAssignmentUpdateRequest, actor ActivityActor) (dto.AdminAssignmentResponse, error)
	Delete(ctx context.Context, id uint, actor ActivityActor) error
	Get(ctx context.Context, id uint) (dto.AdminAssignmentResponse, error)
}

type adminAssignmentService struct {
	repo      repository.AssignmentRepository
	validator *validator.Validate
	activity  ActivityRecorder
	logger    zerolog.Logger
	now       func() time.Time
}

// NewAdminAssignmentService constructs the admin assignment service.
func NewAdminAssignmentService(repo repository.AssignmentRepository, validator *validator.Validate, activity ActivityRecorder, logger zerolog.Logger) AdminAssignmentService {
	return &adminAssignmentService{
		repo:      repo,
		validator: validator,
		activity:  activity,
		logger:    logger.With().Str("component", "admin_assignment_service").Logger(),
		now:       time.Now,
	}
}

func (s *adminAssignmentService) Create(ctx context.Context, payload dto.AdminAssignmentCreateRequest, actor ActivityActor) (dto.AdminAssignmentResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.AdminAssignmentResponse{}, err
	}

	dueDate, err := time.Parse(time.RFC3339, payload.DueDate)
	if err != nil {
		return dto.AdminAssignmentResponse{}, err
	}
	if dueDate.Before(s.now()) {
		return dto.AdminAssignmentResponse{}, ErrAdminAssignmentInvalidDueDate
	}

	assignment := models.Assignment{
		Title:       strings.TrimSpace(payload.Title),
		Description: strings.TrimSpace(payload.Description),
		DueDate:     dueDate,
		FileURL:     strings.TrimSpace(payload.FileURL),
		MaxScore:    payload.MaxScore,
	}
	if payload.Rubric != nil {
		assignment.Rubric = jsonMapFromFloat(payload.Rubric)
	}

	if err := s.repo.Create(ctx, &assignment); err != nil {
		return dto.AdminAssignmentResponse{}, err
	}

	if s.activity != nil {
		metadata := map[string]interface{}{
			"assignment_id": assignment.ID,
			"max_score":     assignment.MaxScore,
			"due_date":      assignment.DueDate,
		}
		_, _ = s.activity.Record(ctx, ActivityEntry{
			ActorID:    actor.ID,
			ActorRole:  actor.Role,
			Action:     "assignment.created",
			EntityType: "assignment",
			EntityID:   &assignment.ID,
			Metadata:   metadata,
		})
	}

	return dto.NewAdminAssignmentResponse(assignment), nil
}

func (s *adminAssignmentService) Update(ctx context.Context, id uint, payload dto.AdminAssignmentUpdateRequest, actor ActivityActor) (dto.AdminAssignmentResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.AdminAssignmentResponse{}, err
	}

	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AdminAssignmentResponse{}, ErrAdminAssignmentNotFound
		}
		return dto.AdminAssignmentResponse{}, err
	}

	changedFields := make([]string, 0)

	if payload.Title != nil {
		assignment.Title = strings.TrimSpace(*payload.Title)
		changedFields = append(changedFields, "title")
	}
	if payload.Description != nil {
		assignment.Description = strings.TrimSpace(*payload.Description)
		changedFields = append(changedFields, "description")
	}
	if payload.DueDate != nil {
		dueDate, err := time.Parse(time.RFC3339, *payload.DueDate)
		if err != nil {
			return dto.AdminAssignmentResponse{}, err
		}
		if dueDate.Before(s.now()) {
			return dto.AdminAssignmentResponse{}, ErrAdminAssignmentInvalidDueDate
		}
		assignment.DueDate = dueDate
		changedFields = append(changedFields, "due_date")
	}
	if payload.MaxScore != nil {
		assignment.MaxScore = *payload.MaxScore
		changedFields = append(changedFields, "max_score")
	}
	if payload.Rubric != nil {
		assignment.Rubric = jsonMapFromFloat(payload.Rubric)
		changedFields = append(changedFields, "rubric")
	}
	if payload.FileURL != nil {
		assignment.FileURL = strings.TrimSpace(*payload.FileURL)
		changedFields = append(changedFields, "file_url")
	}

	if err := s.repo.Update(ctx, &assignment); err != nil {
		return dto.AdminAssignmentResponse{}, err
	}

	if s.activity != nil && len(changedFields) > 0 {
		metadata := map[string]interface{}{
			"assignment_id": assignment.ID,
			"fields":        changedFields,
		}
		if payload.MaxScore != nil {
			metadata["max_score"] = assignment.MaxScore
		}
		if payload.DueDate != nil {
			metadata["due_date"] = assignment.DueDate
		}
		_, _ = s.activity.Record(ctx, ActivityEntry{
			ActorID:    actor.ID,
			ActorRole:  actor.Role,
			Action:     "assignment.updated",
			EntityType: "assignment",
			EntityID:   &assignment.ID,
			Metadata:   metadata,
		})
	}

	return dto.NewAdminAssignmentResponse(assignment), nil
}

func (s *adminAssignmentService) Delete(ctx context.Context, id uint, actor ActivityActor) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminAssignmentNotFound
		}
		return err
	}

	if s.activity != nil {
		_, _ = s.activity.Record(ctx, ActivityEntry{
			ActorID:    actor.ID,
			ActorRole:  actor.Role,
			Action:     "assignment.deleted",
			EntityType: "assignment",
			EntityID:   &id,
			Metadata: map[string]interface{}{
				"assignment_id": id,
			},
		})
	}

	return nil
}

func (s *adminAssignmentService) Get(ctx context.Context, id uint) (dto.AdminAssignmentResponse, error) {
	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AdminAssignmentResponse{}, ErrAdminAssignmentNotFound
		}
		return dto.AdminAssignmentResponse{}, err
	}
	return dto.NewAdminAssignmentResponse(assignment), nil
}

func jsonMapFromFloat(values map[string]float64) datatypes.JSONMap {
	result := datatypes.JSONMap{}
	for key, value := range values {
		result[key] = math.Round(value*100) / 100
	}
	return result
}
