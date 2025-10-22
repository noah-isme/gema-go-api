package service

import (
	"context"
	"errors"
	"strings"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// ErrCodingTaskNotFound indicates the requested coding task does not exist.
var ErrCodingTaskNotFound = errors.New("coding task not found")

// CodingTaskService exposes use cases related to coding tasks.
type CodingTaskService interface {
	List(ctx context.Context, filter dto.CodingTaskFilter) (dto.CodingTaskListResponse, error)
	Get(ctx context.Context, id uint) (dto.CodingTaskDetailResponse, error)
}

type codingTaskService struct {
	repo   repository.CodingTaskRepository
	logger zerolog.Logger
}

// NewCodingTaskService builds a new coding task service.
func NewCodingTaskService(repo repository.CodingTaskRepository, logger zerolog.Logger) CodingTaskService {
	return &codingTaskService{
		repo:   repo,
		logger: logger.With().Str("component", "coding_task_service").Logger(),
	}
}

func (s *codingTaskService) List(ctx context.Context, filter dto.CodingTaskFilter) (dto.CodingTaskListResponse, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	tags := normaliseTags(filter.Tags)
	query := repository.CodingTaskQuery{
		Language:   strings.ToLower(strings.TrimSpace(filter.Language)),
		Difficulty: strings.ToLower(strings.TrimSpace(filter.Difficulty)),
		Tags:       tags,
		Search:     strings.TrimSpace(filter.Search),
		Offset:     (page - 1) * pageSize,
		Limit:      pageSize,
	}

	tasks, total, err := s.repo.List(ctx, query)
	if err != nil {
		return dto.CodingTaskListResponse{}, err
	}

	sanitised := make([]models.CodingTask, 0, len(tasks))
	for _, task := range tasks {
		sanitised = append(sanitised, sanitiseTask(task))
	}

	pagination := dto.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: int(total),
	}

	return dto.NewCodingTaskListResponse(sanitised, pagination), nil
}

func (s *codingTaskService) Get(ctx context.Context, id uint) (dto.CodingTaskDetailResponse, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CodingTaskDetailResponse{}, ErrCodingTaskNotFound
		}
		return dto.CodingTaskDetailResponse{}, err
	}

	return dto.NewCodingTaskDetail(sanitiseTask(task)), nil
}

func normaliseTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.ToLower(strings.TrimSpace(tag))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func sanitiseTask(task models.CodingTask) models.CodingTask {
	task.Prompt = strings.TrimSpace(task.Prompt)
	task.StarterCode = strings.TrimSpace(task.StarterCode)
	task.ExpectedOutput = strings.TrimSpace(task.ExpectedOutput)
	return task
}
