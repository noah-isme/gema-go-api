package service

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

type activityFeedRepo struct {
	items []models.ActivityLog
}

func (r *activityFeedRepo) Create(ctx context.Context, entry *models.ActivityLog) error {
	return nil
}

func (r *activityFeedRepo) List(ctx context.Context, filter repository.ActivityLogFilter) ([]models.ActivityLog, int64, error) {
	return nil, 0, nil
}

func (r *activityFeedRepo) ListRecent(ctx context.Context, filter repository.ActivityLogRecentFilter) ([]models.ActivityLog, int64, error) {
	filtered := make([]models.ActivityLog, 0)
	for _, item := range r.items {
		if filter.ActorID != nil && item.ActorID != *filter.ActorID {
			continue
		}
		if filter.Action != "" && item.Action != filter.Action {
			continue
		}
		if filter.Entity != "" && item.EntityType != filter.Entity {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, int64(len(filtered)), nil
}

func TestActivityFeedServiceCache(t *testing.T) {
	server, err := miniredis.Run()
	require.NoError(t, err)
	defer server.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer redisClient.Close()

	now := time.Now()
	repo := &activityFeedRepo{items: []models.ActivityLog{
		{ID: 1, ActorID: 1, ActorRole: "admin", Action: "create", EntityType: "announcement", CreatedAt: now},
	}}

	svc := NewActivityFeedService(repo, redisClient, time.Minute, testLogger())

	resp, err := svc.ListActive(context.Background(), dto.ActivityFeedRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.False(t, resp.CacheHit)
	require.Len(t, resp.Items, 1)

	// mutate repo to ensure cache keeps previous result
	repo.items = []models.ActivityLog{}

	cached, err := svc.ListActive(context.Background(), dto.ActivityFeedRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.True(t, cached.CacheHit)
	require.Len(t, cached.Items, 1)
}

func TestActivityFeedServiceFilters(t *testing.T) {
	repo := &activityFeedRepo{items: []models.ActivityLog{
		{ID: 1, ActorID: 1, ActorRole: "admin", Action: "create", EntityType: "announcement", CreatedAt: time.Now()},
		{ID: 2, ActorID: 2, ActorRole: "teacher", Action: "update", EntityType: "gallery", CreatedAt: time.Now()},
	}}

	svc := NewActivityFeedService(repo, nil, time.Minute, testLogger())

	userID := uint(2)
	resp, err := svc.ListActive(context.Background(), dto.ActivityFeedRequest{UserID: &userID})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, uint(2), resp.Items[0].ActorID)

	respType, err := svc.ListActive(context.Background(), dto.ActivityFeedRequest{Type: "announcement"})
	require.NoError(t, err)
	require.Len(t, respType.Items, 1)
	require.Equal(t, "announcement", respType.Items[0].EntityType)
}
