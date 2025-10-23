package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/observability"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// AnnouncementService exposes public announcement operations.
type AnnouncementService interface {
	ListActive(ctx context.Context, page, pageSize int) (dto.AnnouncementListResponse, error)
	Seed(ctx context.Context, items []models.Announcement) (int64, error)
}

type announcementService struct {
	repo   repository.AnnouncementRepository
	cache  *redis.Client
	ttl    time.Duration
	logger zerolog.Logger
	policy *bluemonday.Policy
	tracer trace.Tracer
}

// NewAnnouncementService constructs the announcement service.
func NewAnnouncementService(repo repository.AnnouncementRepository, cache *redis.Client, ttl time.Duration, logger zerolog.Logger) AnnouncementService {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("p", "strong", "em", "a", "ul", "ol", "li", "br")
	policy.AllowAttrs("href", "title", "target").OnElements("a")
	return &announcementService{
		repo:   repo,
		cache:  cache,
		ttl:    ttl,
		logger: logger.With().Str("component", "announcement_service").Logger(),
		policy: policy,
		tracer: otel.Tracer("github.com/noah-isme/gema-go-api/internal/service/announcement"),
	}
}

func (s *announcementService) ListActive(ctx context.Context, page, pageSize int) (dto.AnnouncementListResponse, error) {
	ctx, span := s.tracer.Start(ctx, "announcements.fetch", trace.WithAttributes(
		attribute.Int("announcements.page", maxInt(page, 1)),
		attribute.Int("announcements.page_size", clampPageSize(pageSize)),
	))
	defer span.End()

	start := time.Now()
	defer func() {
		observability.AnnouncementsLatency().Observe(time.Since(start).Seconds())
	}()

	page = maxInt(page, 1)
	pageSize = clampPageSize(pageSize)

	span.SetAttributes(
		attribute.Int("announcements.normalized_page", page),
		attribute.Int("announcements.normalized_page_size", pageSize),
	)

	cacheKey := ""
	if s.cache != nil {
		cacheKey = fmt.Sprintf("announcements:active:v1:%d:%d", page, pageSize)
		if cached, err := s.cache.Get(ctx, cacheKey).Result(); err == nil && cached != "" {
			var response dto.AnnouncementListResponse
			if err := json.Unmarshal([]byte(cached), &response); err == nil {
				response.CacheHit = true
				observability.AnnouncementsRequests().WithLabelValues("hit").Inc()
				span.SetAttributes(attribute.String("announcements.cache_status", "hit"))
				span.SetStatus(codes.Ok, "cache hit")
				return response, nil
			}
			span.RecordError(err)
			span.SetAttributes(attribute.String("announcements.cache_status", "corrupt"))
		}
	}

	items, total, err := s.repo.ListActive(ctx, repository.AnnouncementFilter{Page: page, PageSize: pageSize})
	if err != nil {
		observability.AnnouncementsRequests().WithLabelValues("error").Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository failure")
		return dto.AnnouncementListResponse{}, err
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsPinned != items[j].IsPinned {
			return items[i].IsPinned
		}
		return items[i].StartsAt.After(items[j].StartsAt)
	})

	responses := make([]dto.AnnouncementResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, dto.AnnouncementResponse{
			ID:        item.ID,
			Title:     strings.TrimSpace(item.Title),
			Body:      s.policy.Sanitize(item.Body),
			StartsAt:  item.StartsAt,
			EndsAt:    item.EndsAt,
			IsPinned:  item.IsPinned,
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

	response := dto.AnnouncementListResponse{Items: responses, Pagination: pagination}

	if cacheKey != "" && s.cache != nil {
		if payload, err := json.Marshal(response); err == nil {
			if err := s.cache.Set(ctx, cacheKey, payload, s.ttl).Err(); err != nil {
				s.logger.Warn().Err(err).Msg("failed to cache announcements")
				span.RecordError(err)
			}
		} else {
			span.RecordError(err)
		}
	}

	observability.AnnouncementsRequests().WithLabelValues("miss").Inc()
	span.SetAttributes(attribute.String("announcements.cache_status", "miss"), attribute.Int("announcements.total_items", len(responses)))
	span.SetStatus(codes.Ok, "fetched")

	return response, nil
}

func (s *announcementService) Seed(ctx context.Context, items []models.Announcement) (int64, error) {
	affected, err := s.repo.UpsertBatch(ctx, items)
	if err != nil {
		return 0, err
	}
	if s.cache != nil {
		if err := s.cache.FlushDB(ctx).Err(); err != nil {
			s.logger.Warn().Err(err).Msg("failed to flush announcements cache")
		}
	}
	return affected, nil
}
