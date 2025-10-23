package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/observability"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// ActivityFeedService exposes functionality for the public activity stream.
type ActivityFeedService interface {
	ListActive(ctx context.Context, req dto.ActivityFeedRequest) (dto.ActivityFeedResponse, error)
}

type activityFeedService struct {
	repo   repository.ActivityLogRepository
	cache  *redis.Client
	ttl    time.Duration
	logger zerolog.Logger
}

// NewActivityFeedService builds the activity feed service.
func NewActivityFeedService(repo repository.ActivityLogRepository, cache *redis.Client, ttl time.Duration, logger zerolog.Logger) ActivityFeedService {
	if ttl <= 0 {
		ttl = 45 * time.Second
	}
	return &activityFeedService{
		repo:   repo,
		cache:  cache,
		ttl:    ttl,
		logger: logger.With().Str("component", "activity_feed_service").Logger(),
	}
}

func (s *activityFeedService) ListActive(ctx context.Context, req dto.ActivityFeedRequest) (dto.ActivityFeedResponse, error) {
	start := time.Now()
	defer func() {
		observability.ActiveActivitiesLatency().Observe(time.Since(start).Seconds())
	}()

	page := maxInt(req.Page, 1)
	pageSize := clampPageSize(req.PageSize)
	now := time.Now()
	since := now.Add(-24 * time.Hour)

	filter := repository.ActivityLogRecentFilter{
		Since:    since,
		Until:    now,
		Page:     page,
		PageSize: pageSize,
	}

	if req.UserID != nil {
		filter.ActorID = req.UserID
	}

	if trimmed := strings.TrimSpace(req.Action); trimmed != "" {
		filter.Action = strings.ToLower(trimmed)
	}

	if trimmed := strings.TrimSpace(req.Type); trimmed != "" {
		filter.Entity = strings.ToLower(trimmed)
	}

	cacheKey := s.cacheKey(filter)
	if cacheKey != "" && s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey).Result(); err == nil && cached != "" {
			var response dto.ActivityFeedResponse
			if err := json.Unmarshal([]byte(cached), &response); err == nil {
				response.CacheHit = true
				observability.ActiveActivitiesRequests().WithLabelValues("hit").Inc()
				return response, nil
			}
		}
	}

	entries, total, err := s.repo.ListRecent(ctx, filter)
	if err != nil {
		observability.ActiveActivitiesRequests().WithLabelValues("error").Inc()
		return dto.ActivityFeedResponse{}, err
	}

	items := make([]dto.ActivityFeedItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, dto.ActivityFeedItem{
			ID:         entry.ID,
			ActorID:    entry.ActorID,
			ActorRole:  entry.ActorRole,
			Action:     entry.Action,
			EntityType: entry.EntityType,
			EntityID:   entry.EntityID,
			Metadata:   map[string]interface{}(entry.Metadata),
			CreatedAt:  entry.CreatedAt,
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

	response := dto.ActivityFeedResponse{Items: items, Pagination: pagination, CacheHit: false}

	if cacheKey != "" && s.cache != nil {
		if payload, err := json.Marshal(response); err == nil {
			if err := s.cache.Set(ctx, cacheKey, payload, s.ttl).Err(); err != nil {
				s.logger.Warn().Err(err).Msg("failed to write activity feed cache")
			}
		}
	}

	observability.ActiveActivitiesRequests().WithLabelValues("miss").Inc()

	return response, nil
}

func (s *activityFeedService) cacheKey(filter repository.ActivityLogRecentFilter) string {
	if s.cache == nil {
		return ""
	}
	actorKey := "0"
	if filter.ActorID != nil {
		actorKey = fmt.Sprintf("%d", *filter.ActorID)
	}
	return fmt.Sprintf("activities:active:v1:%s:%s:%d:%d:%d", actorKey, filter.Action+"|"+filter.Entity, filter.Page, filter.PageSize, filter.Since.Unix())
}

func clampPageSize(size int) int {
	if size <= 0 {
		return 20
	}
	if size > 100 {
		return 100
	}
	return size
}
