package service

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// ErrAdminStudentNotFound indicates the student was not found for admin operations.
var ErrAdminStudentNotFound = errors.New("admin student not found")

// AdminStudentService orchestrates admin student management use cases.
type AdminStudentService interface {
	List(ctx context.Context, req dto.AdminStudentListRequest) (dto.AdminStudentListResponse, error)
	Get(ctx context.Context, id uint) (dto.AdminStudentResponse, error)
	Update(ctx context.Context, id uint, payload dto.AdminStudentUpdateRequest, actor ActivityActor) (dto.AdminStudentResponse, error)
	Delete(ctx context.Context, id uint, actor ActivityActor) error
}

type adminStudentService struct {
	repo      repository.AdminStudentRepository
	validator *validator.Validate
	activity  ActivityRecorder
	logger    zerolog.Logger
}

// NewAdminStudentService constructs the admin student service.
func NewAdminStudentService(repo repository.AdminStudentRepository, validator *validator.Validate, activity ActivityRecorder, logger zerolog.Logger) AdminStudentService {
	return &adminStudentService{
		repo:      repo,
		validator: validator,
		activity:  activity,
		logger:    logger.With().Str("component", "admin_student_service").Logger(),
	}
}

func (s *adminStudentService) List(ctx context.Context, req dto.AdminStudentListRequest) (dto.AdminStudentListResponse, error) {
	filter := repository.AdminStudentFilter{
		Search:   strings.TrimSpace(req.Search),
		Class:    strings.TrimSpace(req.Class),
		Status:   strings.TrimSpace(req.Status),
		Sort:     req.Sort,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	students, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return dto.AdminStudentListResponse{}, err
	}

	responses := make([]dto.AdminStudentResponse, 0, len(students))
	for _, student := range students {
		responses = append(responses, dto.NewAdminStudentResponse(student))
	}

	pagination := dto.PaginationMeta{
		Page:       maxInt(req.Page, 1),
		PageSize:   req.PageSize,
		TotalItems: total,
	}
	if req.PageSize > 0 {
		pagination.TotalPages = int(math.Ceil(float64(total) / float64(req.PageSize)))
	} else {
		pagination.TotalPages = 1
	}

	return dto.AdminStudentListResponse{Items: responses, Pagination: pagination}, nil
}

func (s *adminStudentService) Get(ctx context.Context, id uint) (dto.AdminStudentResponse, error) {
	student, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AdminStudentResponse{}, ErrAdminStudentNotFound
		}
		return dto.AdminStudentResponse{}, err
	}

	return dto.NewAdminStudentResponse(student), nil
}

func (s *adminStudentService) Update(ctx context.Context, id uint, payload dto.AdminStudentUpdateRequest, actor ActivityActor) (dto.AdminStudentResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.AdminStudentResponse{}, err
	}

	updates := make(map[string]interface{})
	changedFields := make([]string, 0)

	if payload.Name != nil {
		updates["name"] = strings.TrimSpace(*payload.Name)
		changedFields = append(changedFields, "name")
	}
	if payload.Email != nil {
		updates["email"] = strings.TrimSpace(*payload.Email)
		changedFields = append(changedFields, "email")
	}
	if payload.Class != nil {
		updates["class"] = strings.TrimSpace(*payload.Class)
		changedFields = append(changedFields, "class")
	}
	if payload.Status != nil {
		updates["status"] = strings.ToLower(strings.TrimSpace(*payload.Status))
		changedFields = append(changedFields, "status")
	}
	if payload.Flagged != nil {
		updates["flagged"] = *payload.Flagged
		changedFields = append(changedFields, "flagged")
	}
	if payload.Notes != nil {
		updates["notes"] = strings.TrimSpace(*payload.Notes)
		changedFields = append(changedFields, "notes")
	}
	if payload.Flags != nil {
		updates["flags"] = jsonMapFromBool(payload.Flags)
		changedFields = append(changedFields, "flags")
	}

	if len(updates) == 0 {
		student, err := s.repo.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.AdminStudentResponse{}, ErrAdminStudentNotFound
			}
			return dto.AdminStudentResponse{}, err
		}
		return dto.NewAdminStudentResponse(student), nil
	}

	student, err := s.repo.Update(ctx, id, updates)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AdminStudentResponse{}, ErrAdminStudentNotFound
		}
		return dto.AdminStudentResponse{}, err
	}

	metadata := map[string]interface{}{
		"student_id": id,
		"fields":     changedFields,
	}
	if payload.Status != nil {
		metadata["status"] = strings.ToLower(strings.TrimSpace(*payload.Status))
	}

	if s.activity != nil {
		_, _ = s.activity.Record(ctx, ActivityEntry{
			ActorID:    actor.ID,
			ActorRole:  actor.Role,
			Action:     "student.updated",
			EntityType: "student",
			EntityID:   &id,
			Metadata:   metadata,
		})
	}

	return dto.NewAdminStudentResponse(student), nil
}

func (s *adminStudentService) Delete(ctx context.Context, id uint, actor ActivityActor) error {
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminStudentNotFound
		}
		return err
	}

	if s.activity != nil {
		metadata := map[string]interface{}{
			"student_id": id,
			"status":     models.StudentStatusArchived,
		}
		_, _ = s.activity.Record(ctx, ActivityEntry{
			ActorID:    actor.ID,
			ActorRole:  actor.Role,
			Action:     "student.deleted",
			EntityType: "student",
			EntityID:   &id,
			Metadata:   metadata,
		})
	}

	return nil
}

func jsonMapFromBool(flags map[string]bool) datatypes.JSONMap {
	data := datatypes.JSONMap{}
	for key, value := range flags {
		data[key] = value
	}
	return data
}
