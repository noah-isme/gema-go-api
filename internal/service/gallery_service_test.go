package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

type galleryRepoStub struct {
	items []models.GalleryItem
}

func (g *galleryRepoStub) List(ctx context.Context, filter repository.GalleryFilter) ([]models.GalleryItem, int64, error) {
	return g.items, int64(len(g.items)), nil
}

func (g *galleryRepoStub) UpsertBatch(ctx context.Context, items []models.GalleryItem) (int64, error) {
	g.items = items
	return int64(len(items)), nil
}

func TestGalleryServiceList(t *testing.T) {
	repo := &galleryRepoStub{items: []models.GalleryItem{
		{ID: 1, Title: "Sunrise", Caption: "Morning", ImagePath: "sunrise.jpg", Tags: []string{"nature", "sun"}, CreatedAt: time.Now()},
	}}

	svc := NewGalleryService(repo, "https://cdn.example.com/assets", testLogger())

	resp, err := svc.List(context.Background(), []string{"nature"}, "sun", 1, 10)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "https://cdn.example.com/assets/sunrise.jpg", resp.Items[0].ImageURL)
}
