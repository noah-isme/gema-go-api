package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/observability"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// RoadmapService exposes read operations for roadmap stages.
type RoadmapService interface {
	ListStages(ctx context.Context, req dto.RoadmapStageListRequest) (dto.RoadmapStageListResult, error)
}

type roadmapService struct {
	repo   repository.RoadmapStageRepository
	cache  *redis.Client
	ttl    time.Duration
	logger zerolog.Logger
}

// NewRoadmapService constructs the roadmap service.
func NewRoadmapService(repo repository.RoadmapStageRepository, cache *redis.Client, ttl time.Duration, logger zerolog.Logger) RoadmapService {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return &roadmapService{
		repo:   repo,
		cache:  cache,
		ttl:    ttl,
		logger: logger.With().Str("component", "roadmap_service").Logger(),
	}
}

func (s *roadmapService) ListStages(ctx context.Context, req dto.RoadmapStageListRequest) (dto.RoadmapStageListResult, error) {
	start := time.Now()
	defer func() {
		observability.RoadmapLatency().Observe(time.Since(start).Seconds())
	}()

	filter := repository.RoadmapStageFilter{
		Search:   strings.TrimSpace(req.Search),
		Tags:     sanitizeTags(req.Tags),
		Sort:     normalizeRoadmapSort(req.Sort),
		Page:     normalizePage(req.Page),
		PageSize: clampPageSize(req.PageSize),
	}

	if filter.Sort == "" {
		filter.Sort = "sequence"
	}

	if cacheResult, ok := s.fetchCache(ctx, filter); ok {
		cacheResult.CacheHit = true
		observability.RoadmapRequests().WithLabelValues("hit").Inc()
		return cacheResult, nil
	}

	stages, total, err := s.repo.List(ctx, filter)
	if err != nil {
		observability.RoadmapRequests().WithLabelValues("error").Inc()
		return dto.RoadmapStageListResult{}, err
	}

	items := make([]dto.RoadmapStageResponse, 0, len(stages))
	for _, stage := range stages {
		items = append(items, toRoadmapStageResponse(stage))
	}

	pagination := dto.PaginationMeta{
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalItems: total,
		TotalPages: calculateTotalPages(total, filter.PageSize),
	}

	result := dto.RoadmapStageListResult{
		Items:      items,
		Pagination: pagination,
		Filters: dto.RoadmapStageFilters{
			Search: filter.Search,
			Tags:   filter.Tags,
			Sort:   filter.Sort,
		},
	}

	s.writeCache(ctx, filter, result)
	observability.RoadmapRequests().WithLabelValues("miss").Inc()

	return result, nil
}

func (s *roadmapService) fetchCache(ctx context.Context, filter repository.RoadmapStageFilter) (dto.RoadmapStageListResult, bool) {
	if s.cache == nil {
		return dto.RoadmapStageListResult{}, false
	}
	key := s.cacheKey(filter)
	payload, err := s.cache.Get(ctx, key).Result()
	if err != nil {
		return dto.RoadmapStageListResult{}, false
	}

	var result dto.RoadmapStageListResult
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		s.logger.Warn().Err(err).Msg("failed to decode roadmap cache")
		return dto.RoadmapStageListResult{}, false
	}
	return result, true
}

func (s *roadmapService) writeCache(ctx context.Context, filter repository.RoadmapStageFilter, result dto.RoadmapStageListResult) {
	if s.cache == nil {
		return
	}
	key := s.cacheKey(filter)
	payload, err := json.Marshal(result)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to encode roadmap cache")
		return
	}
	if err := s.cache.Set(ctx, key, payload, s.ttl).Err(); err != nil {
		s.logger.Warn().Err(err).Msg("failed to store roadmap cache")
	}
}

func (s *roadmapService) cacheKey(filter repository.RoadmapStageFilter) string {
	tags := strings.Join(filter.Tags, ",")
	return strings.Join([]string{
		"roadmap:v1",
		filter.Sort,
		filter.Search,
		tags,
		intToString(filter.Page),
		intToString(filter.PageSize),
	}, ":")
}

func toRoadmapStageResponse(stage models.RoadmapStage) dto.RoadmapStageResponse {
	return dto.RoadmapStageResponse{
		ID:             stage.ID,
		Slug:           stage.Slug,
		Title:          stage.Title,
		Description:    stage.Description,
		Sequence:       stage.Sequence,
		EstimatedHours: stage.EstimatedHours,
		Icon:           stage.Icon,
		Tags:           append([]string(nil), stage.Tags...),
		Skills:         convertSkills(stage.Skills),
		UpdatedAt:      stage.UpdatedAt,
	}
}

func convertSkills(raw map[string]interface{}) map[string]string {
	if raw == nil {
		return map[string]string{}
	}
	result := make(map[string]string, len(raw))
	for key, value := range raw {
		switch v := value.(type) {
		case string:
			result[key] = v
		case float64:
			result[key] = strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
		default:
			result[key] = fmt.Sprint(v)
		}
	}
	return result
}

func normalizeRoadmapSort(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "sequence", "sequence.asc":
		return "sequence"
	case "-sequence", "sequence.desc":
		return "-sequence"
	case "recent", "updated_at", "updated_at.desc":
		return "recent"
	default:
		return "sequence"
	}
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
