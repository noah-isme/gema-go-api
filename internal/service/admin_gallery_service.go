package service

import (
	"context"
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// AdminGalleryService exposes admin gallery management use cases.
type AdminGalleryService interface {
	List(ctx context.Context, req dto.AdminGalleryListRequest) (dto.AdminGalleryListResponse, error)
	Create(ctx context.Context, payload dto.AdminGalleryRequest, actor ActivityActor) (dto.AdminGalleryResponse, error)
	Update(ctx context.Context, id uint, payload dto.AdminGalleryRequest, actor ActivityActor) (dto.AdminGalleryResponse, error)
	Delete(ctx context.Context, id uint, actor ActivityActor) error
}

// ErrAdminGalleryNotFound indicates gallery entry missing.
var ErrAdminGalleryNotFound = errors.New("gallery item not found")

type adminGalleryService struct {
	repo      repository.GalleryRepository
	validator *validator.Validate
	activity  ActivityRecorder
	logger    zerolog.Logger
}

// NewAdminGalleryService constructs the gallery admin service.
func NewAdminGalleryService(repo repository.GalleryRepository, validator *validator.Validate, activity ActivityRecorder, logger zerolog.Logger) AdminGalleryService {
	return &adminGalleryService{
		repo:      repo,
		validator: validator,
		activity:  activity,
		logger:    logger.With().Str("component", "admin_gallery_service").Logger(),
	}
}

func (s *adminGalleryService) List(ctx context.Context, req dto.AdminGalleryListRequest) (dto.AdminGalleryListResponse, error) {
	filter := repository.GalleryFilter{
		Tags:     sanitizeTags(req.Tags),
		Search:   strings.TrimSpace(req.Search),
		Page:     normalizePage(req.Page),
		PageSize: clampPageSize(req.PageSize),
	}

	items, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return dto.AdminGalleryListResponse{}, err
	}

	responses := make([]dto.AdminGalleryResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, toAdminGalleryResponse(item))
	}

	pagination := dto.PaginationMeta{
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalItems: total,
		TotalPages: calculateTotalPages(total, filter.PageSize),
	}

	return dto.AdminGalleryListResponse{Items: responses, Pagination: pagination}, nil
}

func (s *adminGalleryService) Create(ctx context.Context, payload dto.AdminGalleryRequest, actor ActivityActor) (dto.AdminGalleryResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.AdminGalleryResponse{}, err
	}

	item := models.GalleryItem{
		Slug:      generateContentSlug(payload.Title),
		Title:     strings.TrimSpace(payload.Title),
		Caption:   strings.TrimSpace(payload.Caption),
		ImagePath: strings.TrimSpace(payload.ImageURL),
		Tags:      sanitizeTags(payload.Tags),
	}

	if err := s.repo.Create(ctx, &item); err != nil {
		return dto.AdminGalleryResponse{}, err
	}

	s.recordActivity(ctx, actor, "gallery.created", item.ID)
	return toAdminGalleryResponse(item), nil
}

func (s *adminGalleryService) Update(ctx context.Context, id uint, payload dto.AdminGalleryRequest, actor ActivityActor) (dto.AdminGalleryResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.AdminGalleryResponse{}, err
	}

	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AdminGalleryResponse{}, ErrAdminGalleryNotFound
		}
		return dto.AdminGalleryResponse{}, err
	}

	item.Title = strings.TrimSpace(payload.Title)
	item.Caption = strings.TrimSpace(payload.Caption)
	item.ImagePath = strings.TrimSpace(payload.ImageURL)
	item.Tags = sanitizeTags(payload.Tags)

	if err := s.repo.Update(ctx, &item); err != nil {
		return dto.AdminGalleryResponse{}, err
	}

	s.recordActivity(ctx, actor, "gallery.updated", item.ID)
	return toAdminGalleryResponse(item), nil
}

func (s *adminGalleryService) Delete(ctx context.Context, id uint, actor ActivityActor) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminGalleryNotFound
		}
		return err
	}
	s.recordActivity(ctx, actor, "gallery.deleted", id)
	return nil
}

func (s *adminGalleryService) recordActivity(ctx context.Context, actor ActivityActor, action string, id uint) {
	if s.activity == nil {
		return
	}
	entry := ActivityEntry{
		ActorID:    actor.ID,
		ActorRole:  actor.Role,
		Action:     action,
		EntityType: "gallery",
		EntityID:   &id,
	}
	if _, err := s.activity.Record(ctx, entry); err != nil {
		s.logger.Warn().Err(err).Msg("failed to record gallery activity")
	}
}

func toAdminGalleryResponse(item models.GalleryItem) dto.AdminGalleryResponse {
	return dto.AdminGalleryResponse{
		ID:        item.ID,
		Slug:      item.Slug,
		Title:     item.Title,
		Caption:   item.Caption,
		ImageURL:  item.ImagePath,
		Tags:      append([]string(nil), item.Tags...),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
