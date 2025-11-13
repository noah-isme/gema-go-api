package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// AdminAnnouncementService handles admin announcement flows.
type AdminAnnouncementService interface {
	List(ctx context.Context, req dto.AdminAnnouncementListRequest) (dto.AdminAnnouncementListResponse, error)
	Create(ctx context.Context, payload dto.AdminAnnouncementRequest, actor ActivityActor) (dto.AdminAnnouncementResponse, error)
}

type adminAnnouncementService struct {
	repo      repository.AnnouncementRepository
	validator *validator.Validate
	cache     *redis.Client
	activity  ActivityRecorder
	logger    zerolog.Logger
}

// ErrAdminAnnouncementNotFound indicates announcement missing.
var ErrAdminAnnouncementNotFound = errors.New("announcement not found")

// NewAdminAnnouncementService constructs the service.
func NewAdminAnnouncementService(repo repository.AnnouncementRepository, cache *redis.Client, validator *validator.Validate, activity ActivityRecorder, logger zerolog.Logger) AdminAnnouncementService {
	return &adminAnnouncementService{
		repo:      repo,
		validator: validator,
		cache:     cache,
		activity:  activity,
		logger:    logger.With().Str("component", "admin_announcement_service").Logger(),
	}
}

func (s *adminAnnouncementService) List(ctx context.Context, req dto.AdminAnnouncementListRequest) (dto.AdminAnnouncementListResponse, error) {
	filter := repository.AdminAnnouncementFilter{
		Page:     normalizePage(req.Page),
		PageSize: clampPageSize(req.PageSize),
		Search:   strings.TrimSpace(req.Search),
	}

	items, total, err := s.repo.ListAll(ctx, filter)
	if err != nil {
		return dto.AdminAnnouncementListResponse{}, err
	}

	responses := make([]dto.AdminAnnouncementResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, toAdminAnnouncementResponse(item))
	}

	pagination := dto.PaginationMeta{
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalItems: total,
		TotalPages: calculateTotalPages(total, filter.PageSize),
	}

	return dto.AdminAnnouncementListResponse{Items: responses, Pagination: pagination}, nil
}

func (s *adminAnnouncementService) Create(ctx context.Context, payload dto.AdminAnnouncementRequest, actor ActivityActor) (dto.AdminAnnouncementResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.AdminAnnouncementResponse{}, err
	}

	startsAt, err := time.Parse(time.RFC3339, payload.StartsAt)
	if err != nil {
		return dto.AdminAnnouncementResponse{}, err
	}

	var endsAt *time.Time
	if strings.TrimSpace(payload.EndsAt) != "" {
		parsed, parseErr := time.Parse(time.RFC3339, payload.EndsAt)
		if parseErr != nil {
			return dto.AdminAnnouncementResponse{}, parseErr
		}
		endsAt = &parsed
	}

	model := models.Announcement{
		Slug:     generateContentSlug(payload.Title),
		Title:    strings.TrimSpace(payload.Title),
		Body:     strings.TrimSpace(payload.Body),
		StartsAt: startsAt,
		EndsAt:   endsAt,
		IsPinned: payload.IsPinned,
	}

	if err := s.repo.Create(ctx, &model); err != nil {
		return dto.AdminAnnouncementResponse{}, err
	}

	if s.cache != nil {
		if err := s.cache.FlushDB(ctx).Err(); err != nil {
			s.logger.Warn().Err(err).Msg("failed to flush announcement cache")
		}
	}

	if s.activity != nil {
		s.activity.Record(ctx, ActivityEntry{
			ActorID:    actor.ID,
			ActorRole:  actor.Role,
			Action:     "announcement.created",
			EntityType: "announcement",
			EntityID:   &model.ID,
		})
	}

	return toAdminAnnouncementResponse(model), nil
}

func toAdminAnnouncementResponse(model models.Announcement) dto.AdminAnnouncementResponse {
	return dto.AdminAnnouncementResponse{
		ID:        model.ID,
		Slug:      model.Slug,
		Title:     model.Title,
		Body:      model.Body,
		StartsAt:  model.StartsAt,
		EndsAt:    model.EndsAt,
		IsPinned:  model.IsPinned,
		CreatedAt: model.CreatedAt,
	}
}
