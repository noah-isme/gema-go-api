package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

var (
	// ErrTutorialArticleNotFound indicates article lookup failed.
	ErrTutorialArticleNotFound = errors.New("tutorial article not found")
	// ErrTutorialProjectNotFound indicates project lookup failed.
	ErrTutorialProjectNotFound = errors.New("tutorial project not found")
)

// TutorialContentService exposes tutorial articles & projects.
type TutorialContentService interface {
	ListArticles(ctx context.Context, req dto.TutorialContentListRequest) (dto.TutorialArticleListResult, error)
	ListProjects(ctx context.Context, req dto.TutorialContentListRequest) (dto.TutorialProjectListResult, error)
	GetArticle(ctx context.Context, id uint) (dto.TutorialArticleResponse, error)
	GetProject(ctx context.Context, id uint) (dto.TutorialProjectResponse, error)
	CreateArticle(ctx context.Context, payload dto.TutorialArticleCreateRequest) (dto.TutorialArticleResponse, error)
	CreateProject(ctx context.Context, payload dto.TutorialProjectCreateRequest) (dto.TutorialProjectResponse, error)
}

type tutorialContentService struct {
	articles  repository.TutorialArticleRepository
	projects  repository.TutorialProjectRepository
	validator *validator.Validate
	logger    zerolog.Logger
	now       func() time.Time
}

// NewTutorialContentService constructs the tutorial content service.
func NewTutorialContentService(
	articleRepo repository.TutorialArticleRepository,
	projectRepo repository.TutorialProjectRepository,
	validate *validator.Validate,
	logger zerolog.Logger,
) TutorialContentService {
	return &tutorialContentService{
		articles:  articleRepo,
		projects:  projectRepo,
		validator: validate,
		logger:    logger.With().Str("component", "tutorial_content_service").Logger(),
		now:       time.Now,
	}
}

func (s *tutorialContentService) ListArticles(ctx context.Context, req dto.TutorialContentListRequest) (dto.TutorialArticleListResult, error) {
	filter := s.buildFilter(req)
	records, total, err := s.articles.List(ctx, filter)
	if err != nil {
		return dto.TutorialArticleListResult{}, err
	}

	items := make([]dto.TutorialArticleResponse, 0, len(records))
	for _, record := range records {
		items = append(items, dto.NewTutorialArticleResponse(record))
	}

	pagination := dto.PaginationMeta{
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalItems: total,
		TotalPages: calculateTotalPages(total, filter.PageSize),
	}

	return dto.TutorialArticleListResult{
		Items:      items,
		Pagination: pagination,
		Filters: dto.TutorialContentFilters{
			Tags:   filter.Tags,
			Search: filter.Search,
			Sort:   filter.Sort,
		},
	}, nil
}

func (s *tutorialContentService) ListProjects(ctx context.Context, req dto.TutorialContentListRequest) (dto.TutorialProjectListResult, error) {
	filter := s.buildFilter(req)
	records, total, err := s.projects.List(ctx, filter)
	if err != nil {
		return dto.TutorialProjectListResult{}, err
	}

	items := make([]dto.TutorialProjectResponse, 0, len(records))
	for _, record := range records {
		items = append(items, dto.NewTutorialProjectResponse(record))
	}

	pagination := dto.PaginationMeta{
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalItems: total,
		TotalPages: calculateTotalPages(total, filter.PageSize),
	}

	return dto.TutorialProjectListResult{
		Items:      items,
		Pagination: pagination,
		Filters: dto.TutorialContentFilters{
			Tags:   filter.Tags,
			Search: filter.Search,
			Sort:   filter.Sort,
		},
	}, nil
}

func (s *tutorialContentService) GetArticle(ctx context.Context, id uint) (dto.TutorialArticleResponse, error) {
	record, err := s.articles.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.TutorialArticleResponse{}, ErrTutorialArticleNotFound
		}
		return dto.TutorialArticleResponse{}, err
	}
	return dto.NewTutorialArticleResponse(record), nil
}

func (s *tutorialContentService) GetProject(ctx context.Context, id uint) (dto.TutorialProjectResponse, error) {
	record, err := s.projects.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.TutorialProjectResponse{}, ErrTutorialProjectNotFound
		}
		return dto.TutorialProjectResponse{}, err
	}
	return dto.NewTutorialProjectResponse(record), nil
}

func (s *tutorialContentService) CreateArticle(ctx context.Context, payload dto.TutorialArticleCreateRequest) (dto.TutorialArticleResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.TutorialArticleResponse{}, err
	}

	now := s.now()
	article := models.TutorialArticle{
		Slug:           generateContentSlug(payload.Title),
		Title:          strings.TrimSpace(payload.Title),
		Summary:        strings.TrimSpace(payload.Summary),
		Content:        strings.TrimSpace(payload.Content),
		Tags:           sanitizeTags(payload.Tags),
		ThumbnailURL:   strings.TrimSpace(payload.ThumbnailURL),
		Author:         strings.TrimSpace(payload.Author),
		ReadingMinutes: normalizeReadingMinutes(payload.ReadingMinutes),
		Status:         strings.ToLower(strings.TrimSpace(payload.Status)),
	}

	if article.Status == "published" {
		article.PublishedAt = &now
	}

	if err := s.articles.Create(ctx, &article); err != nil {
		return dto.TutorialArticleResponse{}, err
	}

	return dto.NewTutorialArticleResponse(article), nil
}

func (s *tutorialContentService) CreateProject(ctx context.Context, payload dto.TutorialProjectCreateRequest) (dto.TutorialProjectResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.TutorialProjectResponse{}, err
	}

	project := models.TutorialProject{
		Slug:           generateContentSlug(payload.Title),
		Title:          strings.TrimSpace(payload.Title),
		Summary:        strings.TrimSpace(payload.Summary),
		Content:        strings.TrimSpace(payload.Content),
		Difficulty:     strings.ToLower(strings.TrimSpace(payload.Difficulty)),
		EstimatedHours: normalizeEstimatedHours(payload.EstimatedHours),
		Tags:           sanitizeTags(payload.Tags),
		RepoURL:        strings.TrimSpace(payload.RepoURL),
		PreviewURL:     strings.TrimSpace(payload.PreviewURL),
		Status:         strings.ToLower(strings.TrimSpace(payload.Status)),
	}

	if err := s.projects.Create(ctx, &project); err != nil {
		return dto.TutorialProjectResponse{}, err
	}

	return dto.NewTutorialProjectResponse(project), nil
}

func (s *tutorialContentService) buildFilter(req dto.TutorialContentListRequest) repository.TutorialContentFilter {
	page := normalizePage(req.Page)
	pageSize := clampPageSize(req.PageSize)
	tags := sanitizeTags(req.Tags)
	sort := strings.ToLower(strings.TrimSpace(req.Sort))
	if sort == "" {
		sort = "recent"
	}

	return repository.TutorialContentFilter{
		Search:   strings.TrimSpace(req.Search),
		Tags:     tags,
		Sort:     sort,
		Page:     page,
		PageSize: pageSize,
	}
}

func sanitizeTags(tags []string) []string {
	cleaned := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		normalized := strings.ToLower(strings.TrimSpace(tag))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		cleaned = append(cleaned, normalized)
	}
	return cleaned
}

func normalizeReadingMinutes(value int) int {
	if value <= 0 {
		return 5
	}
	if value > 300 {
		return 300
	}
	return value
}

func normalizeEstimatedHours(value int) int {
	if value <= 0 {
		return 2
	}
	if value > 200 {
		return 200
	}
	return value
}

func generateContentSlug(title string) string {
	base := strings.ToLower(strings.TrimSpace(title))
	if base == "" {
		base = "content"
	}

	slug := make([]rune, 0, len(base))
	for _, r := range base {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			slug = append(slug, r)
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if len(slug) == 0 || slug[len(slug)-1] == '-' {
				continue
			}
			slug = append(slug, '-')
		}
	}
	trimmed := strings.Trim(string(slug), "-")
	if trimmed == "" {
		trimmed = "content"
	}
	return trimmed + "-" + uuid.NewString()[:8]
}
