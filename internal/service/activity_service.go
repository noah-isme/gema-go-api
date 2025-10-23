package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gorm.io/datatypes"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// ActivityActor represents the authenticated actor performing an admin action.
type ActivityActor struct {
	ID   uint
	Role string
}

// ActivityEntry captures the details required to persist an audit entry.
type ActivityEntry struct {
	ActorID    uint
	ActorRole  string
	Action     string
	EntityType string
	EntityID   *uint
	Metadata   map[string]interface{}
}

// ActivityRecorder defines behaviour for recording activity logs.
type ActivityRecorder interface {
	Record(ctx context.Context, entry ActivityEntry) (dto.AdminActivityResponse, error)
}

// ActivityService exposes methods to query and persist activity logs.
type ActivityService interface {
	ActivityRecorder
	List(ctx context.Context, req dto.AdminActivityListRequest) (dto.AdminActivityListResponse, error)
	Create(ctx context.Context, actor ActivityActor, payload dto.AdminActivityCreateRequest) (dto.AdminActivityResponse, error)
}

type activityService struct {
	repo      repository.ActivityLogRepository
	validator *validator.Validate
	logger    zerolog.Logger
}

// NewActivityService constructs the activity log service.
func NewActivityService(repo repository.ActivityLogRepository, validator *validator.Validate, logger zerolog.Logger) ActivityService {
	return &activityService{
		repo:      repo,
		validator: validator,
		logger:    logger.With().Str("component", "activity_service").Logger(),
	}
}

func (s *activityService) Create(ctx context.Context, actor ActivityActor, payload dto.AdminActivityCreateRequest) (dto.AdminActivityResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.AdminActivityResponse{}, err
	}

	entry := ActivityEntry{
		ActorID:    actor.ID,
		ActorRole:  actor.Role,
		Action:     payload.Action,
		EntityType: payload.EntityType,
		EntityID:   payload.EntityID,
		Metadata:   payload.Metadata,
	}

	return s.Record(ctx, entry)
}

func (s *activityService) Record(ctx context.Context, entry ActivityEntry) (dto.AdminActivityResponse, error) {
	if strings.TrimSpace(entry.Action) == "" {
		return dto.AdminActivityResponse{}, fmt.Errorf("action is required")
	}
	if strings.TrimSpace(entry.EntityType) == "" {
		return dto.AdminActivityResponse{}, fmt.Errorf("entity type is required")
	}

	actorRole := normalizeRole(entry.ActorRole)
	model := models.ActivityLog{
		ActorID:    entry.ActorID,
		ActorRole:  actorRole,
		Action:     strings.ToLower(strings.TrimSpace(entry.Action)),
		EntityType: strings.ToLower(strings.TrimSpace(entry.EntityType)),
		EntityID:   entry.EntityID,
		Metadata:   sanitizeMetadata(entry.Metadata),
	}

	if err := s.repo.Create(ctx, &model); err != nil {
		s.logger.Error().Err(err).Msg("failed to persist activity log")
		return dto.AdminActivityResponse{}, err
	}

	return dto.NewAdminActivityResponse(model), nil
}

func (s *activityService) List(ctx context.Context, req dto.AdminActivityListRequest) (dto.AdminActivityListResponse, error) {
	filter := repository.ActivityLogFilter{
		Page:       req.Page,
		PageSize:   req.PageSize,
		Action:     strings.TrimSpace(req.Action),
		EntityType: strings.TrimSpace(req.EntityType),
	}
	if req.ActorID > 0 {
		filter.ActorID = &req.ActorID
	}

	entries, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return dto.AdminActivityListResponse{}, err
	}

	responses := make([]dto.AdminActivityResponse, 0, len(entries))
	for _, entry := range entries {
		responses = append(responses, dto.NewAdminActivityResponse(entry))
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

	return dto.AdminActivityListResponse{Items: responses, Pagination: pagination}, nil
}

func sanitizeMetadata(metadata map[string]interface{}) datatypes.JSONMap {
	if metadata == nil {
		return datatypes.JSONMap{}
	}

	sanitized := datatypes.JSONMap{}
	for key, value := range metadata {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "email") || strings.Contains(lower, "token") {
			sanitized[key] = "***"
			continue
		}
		sanitized[key] = value
	}
	return sanitized
}

func normalizeRole(role string) string {
	r := strings.ToLower(strings.TrimSpace(role))
	if r == "" {
		return "system"
	}
	return r
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
