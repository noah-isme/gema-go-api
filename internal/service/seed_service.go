package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

var (
	// ErrSeedDisabled indicates the seeding tools are disabled by configuration.
	ErrSeedDisabled = errors.New("seeding is disabled")
	// ErrSeedUnauthorized indicates the provided token is invalid.
	ErrSeedUnauthorized = errors.New("invalid seed token")
)

// SeedService orchestrates content seeding operations.
type SeedService interface {
	SeedAnnouncements(ctx context.Context, token string, items []models.Announcement) (int64, error)
	SeedGallery(ctx context.Context, token string, items []models.GalleryItem) (int64, error)
}

type seedService struct {
	announcementRepo repository.AnnouncementRepository
	galleryRepo      repository.GalleryRepository
	enabled          bool
	token            string
	logger           zerolog.Logger
}

// NewSeedService constructs a seeding service.
func NewSeedService(announcementRepo repository.AnnouncementRepository, galleryRepo repository.GalleryRepository, enabled bool, token string, logger zerolog.Logger) SeedService {
	return &seedService{
		announcementRepo: announcementRepo,
		galleryRepo:      galleryRepo,
		enabled:          enabled,
		token:            token,
		logger:           logger.With().Str("component", "seed_service").Logger(),
	}
}

func (s *seedService) SeedAnnouncements(ctx context.Context, token string, items []models.Announcement) (int64, error) {
	if !s.enabled {
		return 0, ErrSeedDisabled
	}
	if !s.validateToken(token) {
		return 0, ErrSeedUnauthorized
	}
	normalized := normalizeAnnouncements(items)
	affected, err := s.announcementRepo.UpsertBatch(ctx, normalized)
	if err != nil {
		return 0, err
	}
	s.logger.Info().Int64("affected", affected).Msg("announcements seeded")
	return affected, nil
}

func (s *seedService) SeedGallery(ctx context.Context, token string, items []models.GalleryItem) (int64, error) {
	if !s.enabled {
		return 0, ErrSeedDisabled
	}
	if !s.validateToken(token) {
		return 0, ErrSeedUnauthorized
	}
	normalized := normalizeGallery(items)
	affected, err := s.galleryRepo.UpsertBatch(ctx, normalized)
	if err != nil {
		return 0, err
	}
	s.logger.Info().Int64("affected", affected).Msg("gallery seeded")
	return affected, nil
}

func (s *seedService) validateToken(token string) bool {
	expected := strings.TrimSpace(s.token)
	if expected == "" {
		return false
	}
	return subtleConstantTimeCompare(expected, strings.TrimSpace(token))
}

func subtleConstantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	mismatch := byte(0)
	for i := 0; i < len(a); i++ {
		mismatch |= a[i] ^ b[i]
	}
	return mismatch == 0
}

func normalizeAnnouncements(items []models.Announcement) []models.Announcement {
	now := time.Now()
	for i := range items {
		if items[i].StartsAt.IsZero() {
			items[i].StartsAt = now
		}
		if items[i].Slug == "" {
			items[i].Slug = strings.ReplaceAll(strings.ToLower(items[i].Title), " ", "-")
		}
	}
	return items
}

func normalizeGallery(items []models.GalleryItem) []models.GalleryItem {
	for i := range items {
		if items[i].Slug == "" {
			items[i].Slug = strings.ReplaceAll(strings.ToLower(items[i].Title), " ", "-")
		}
	}
	return items
}
