package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

type seedAnnRepo struct {
	items []models.Announcement
}

func (s *seedAnnRepo) ListActive(ctx context.Context, filter repository.AnnouncementFilter) ([]models.Announcement, int64, error) {
	return nil, 0, nil
}

func (s *seedAnnRepo) UpsertBatch(ctx context.Context, items []models.Announcement) (int64, error) {
	s.items = items
	return int64(len(items)), nil
}

type seedGalleryRepo struct {
	items []models.GalleryItem
}

func (s *seedGalleryRepo) List(ctx context.Context, filter repository.GalleryFilter) ([]models.GalleryItem, int64, error) {
	return nil, 0, nil
}

func (s *seedGalleryRepo) UpsertBatch(ctx context.Context, items []models.GalleryItem) (int64, error) {
	s.items = items
	return int64(len(items)), nil
}

func TestSeedServiceTokenGuard(t *testing.T) {
	annRepo := &seedAnnRepo{}
	galRepo := &seedGalleryRepo{}
	svc := NewSeedService(annRepo, galRepo, true, "secret", testLogger())

	_, err := svc.SeedAnnouncements(context.Background(), "wrong", []models.Announcement{{Title: "Test"}})
	require.ErrorIs(t, err, ErrSeedUnauthorized)

	affected, err := svc.SeedAnnouncements(context.Background(), "secret", []models.Announcement{{Title: "Test", StartsAt: time.Now()}})
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
}
