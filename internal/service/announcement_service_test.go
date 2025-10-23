package service

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

type announcementRepoStub struct {
	items []models.Announcement
}

func (a *announcementRepoStub) ListActive(ctx context.Context, filter repository.AnnouncementFilter) ([]models.Announcement, int64, error) {
	return a.items, int64(len(a.items)), nil
}

func (a *announcementRepoStub) UpsertBatch(ctx context.Context, items []models.Announcement) (int64, error) {
	a.items = items
	return int64(len(items)), nil
}

func TestAnnouncementServiceCachingAndSanitize(t *testing.T) {
	server, err := miniredis.Run()
	require.NoError(t, err)
	defer server.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer redisClient.Close()

	repo := &announcementRepoStub{items: []models.Announcement{{
		ID:       1,
		Title:    "Hello",
		Body:     "<script>alert('x')</script><p>Safe</p>",
		StartsAt: time.Now().Add(-time.Hour),
		EndsAt:   nil,
		IsPinned: false,
	}}}

	svc := NewAnnouncementService(repo, redisClient, time.Minute, testLogger())

	resp, err := svc.ListActive(context.Background(), 1, 10)
	require.NoError(t, err)
	require.False(t, resp.CacheHit)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "Hello", resp.Items[0].Title)
	require.Equal(t, "<p>Safe</p>", resp.Items[0].Body)

	repo.items = nil
	cached, err := svc.ListActive(context.Background(), 1, 10)
	require.NoError(t, err)
	require.True(t, cached.CacheHit)
	require.Len(t, cached.Items, 1)
}

func TestAnnouncementServicePinnedOrdering(t *testing.T) {
	repo := &announcementRepoStub{items: []models.Announcement{
		{ID: 1, Title: "Scheduled", Body: "ok", StartsAt: time.Now().Add(-time.Hour)},
		{ID: 2, Title: "Pinned", Body: "ok", StartsAt: time.Now().Add(-48 * time.Hour), IsPinned: true},
	}}

	svc := NewAnnouncementService(repo, nil, time.Minute, testLogger())

	resp, err := svc.ListActive(context.Background(), 1, 10)
	require.NoError(t, err)
	require.Len(t, resp.Items, 2)
	require.Equal(t, "Pinned", resp.Items[0].Title)
}
