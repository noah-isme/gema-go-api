package service

import (
	"context"
	"math"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/observability"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// GalleryService exposes read operations for the public gallery.
type GalleryService interface {
	List(ctx context.Context, tags []string, search string, page, pageSize int) (dto.GalleryListResponse, error)
	Seed(ctx context.Context, items repository.GalleryFilter, upsert func(ctx context.Context) error) error
}

type galleryService struct {
	repo    repository.GalleryRepository
	logger  zerolog.Logger
	cdnBase string
}

// NewGalleryService constructs the gallery service.
func NewGalleryService(repo repository.GalleryRepository, cdnBase string, logger zerolog.Logger) GalleryService {
	return &galleryService{
		repo:    repo,
		logger:  logger.With().Str("component", "gallery_service").Logger(),
		cdnBase: strings.TrimRight(cdnBase, "/"),
	}
}

func (s *galleryService) List(ctx context.Context, tags []string, search string, page, pageSize int) (dto.GalleryListResponse, error) {
	start := time.Now()
	defer func() {
		observability.GalleryLatency().Observe(time.Since(start).Seconds())
	}()

	page = maxInt(page, 1)
	pageSize = clampPageSize(pageSize)

	filter := repository.GalleryFilter{Tags: tags, Search: search, Page: page, PageSize: pageSize}
	items, total, err := s.repo.List(ctx, filter)
	if err != nil {
		observability.GalleryRequests().WithLabelValues("error").Inc()
		return dto.GalleryListResponse{}, err
	}

	responses := make([]dto.GalleryItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, dto.GalleryItemResponse{
			ID:        item.ID,
			Title:     item.Title,
			Caption:   item.Caption,
			ImageURL:  s.normalizeURL(item.ImagePath),
			Tags:      append([]string(nil), item.Tags...),
			CreatedAt: item.CreatedAt,
		})
	}

	pagination := dto.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
	}
	if pageSize > 0 {
		pagination.TotalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	} else {
		pagination.TotalPages = 1
	}

	observability.GalleryRequests().WithLabelValues("success").Inc()

	return dto.GalleryListResponse{Items: responses, Pagination: pagination}, nil
}

func (s *galleryService) Seed(ctx context.Context, items repository.GalleryFilter, upsert func(ctx context.Context) error) error {
	return upsert(ctx)
}

func (s *galleryService) normalizeURL(imagePath string) string {
	trimmed := strings.TrimSpace(imagePath)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	if s.cdnBase == "" {
		return trimmed
	}
	base, err := url.Parse(s.cdnBase)
	if err != nil {
		s.logger.Warn().Err(err).Str("base", s.cdnBase).Msg("invalid gallery CDN base")
		return trimmed
	}
	base.Path = path.Join(base.Path, trimmed)
	return base.String()
}
