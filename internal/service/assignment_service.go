package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// ErrAssignmentNotFound indicates the requested assignment does not exist.
var ErrAssignmentNotFound = errors.New("assignment not found")

// FileUploader abstracts uploading binary data and returning a URL.
type FileUploader interface {
	Upload(ctx context.Context, name string, reader io.Reader) (string, error)
}

// AssignmentService exposes assignment domain use cases.
type AssignmentService interface {
	List(ctx context.Context, req dto.AssignmentListRequest) (dto.AssignmentListResult, error)
	Get(ctx context.Context, id uint) (dto.AssignmentResponse, error)
	Create(ctx context.Context, payload dto.AssignmentCreateRequest, file *multipart.FileHeader) (dto.AssignmentResponse, error)
	Update(ctx context.Context, id uint, payload dto.AssignmentUpdateRequest, file *multipart.FileHeader) (dto.AssignmentResponse, error)
	Delete(ctx context.Context, id uint) error
}

type assignmentService struct {
	repo      repository.AssignmentRepository
	validator *validator.Validate
	uploader  FileUploader
	logger    zerolog.Logger
	now       func() time.Time
}

// NewAssignmentService builds a new assignment service.
func NewAssignmentService(repo repository.AssignmentRepository, validate *validator.Validate, uploader FileUploader, logger zerolog.Logger) AssignmentService {
	return &assignmentService{
		repo:      repo,
		validator: validate,
		uploader:  uploader,
		logger:    logger.With().Str("component", "assignment_service").Logger(),
		now:       time.Now,
	}
}

func (s *assignmentService) List(ctx context.Context, req dto.AssignmentListRequest) (dto.AssignmentListResult, error) {
	page := normalizePage(req.Page)
	pageSize := clampPageSize(req.PageSize)
	search := strings.TrimSpace(req.Search)
	sort := strings.ToLower(strings.TrimSpace(req.Sort))
	if sort == "" {
		sort = "due_date"
	}

	filter := repository.AssignmentFilter{
		Search:   search,
		Sort:     sort,
		Page:     page,
		PageSize: pageSize,
	}

	assignments, total, err := s.repo.ListWithFilter(ctx, filter)
	if err != nil {
		return dto.AssignmentListResult{}, err
	}

	items := dto.NewAssignmentResponseSlice(assignments)
	pagination := dto.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: calculateTotalPages(total, pageSize),
	}

	return dto.AssignmentListResult{
		Items:      items,
		Pagination: pagination,
		Sort:       sort,
		Search:     search,
	}, nil
}

func (s *assignmentService) Get(ctx context.Context, id uint) (dto.AssignmentResponse, error) {
	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AssignmentResponse{}, ErrAssignmentNotFound
		}

		return dto.AssignmentResponse{}, err
	}

	return dto.NewAssignmentResponse(assignment), nil
}

func (s *assignmentService) Create(ctx context.Context, payload dto.AssignmentCreateRequest, file *multipart.FileHeader) (dto.AssignmentResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.AssignmentResponse{}, err
	}

	dueDate, err := time.Parse(time.RFC3339, payload.DueDate)
	if err != nil {
		return dto.AssignmentResponse{}, fmt.Errorf("invalid due date: %w", err)
	}

	if !dueDate.After(s.now()) {
		return dto.AssignmentResponse{}, fmt.Errorf("due date must be in the future")
	}

	assignment := models.Assignment{
		Title:       payload.Title,
		Description: payload.Description,
		DueDate:     dueDate,
	}

	if file != nil {
		url, err := s.uploadFile(ctx, file)
		if err != nil {
			return dto.AssignmentResponse{}, err
		}
		assignment.FileURL = url
	}

	if err := s.repo.Create(ctx, &assignment); err != nil {
		return dto.AssignmentResponse{}, err
	}

	s.logger.Info().Uint("assignment_id", assignment.ID).Msg("assignment created")

	return dto.NewAssignmentResponse(assignment), nil
}

func (s *assignmentService) Update(ctx context.Context, id uint, payload dto.AssignmentUpdateRequest, file *multipart.FileHeader) (dto.AssignmentResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.AssignmentResponse{}, err
	}

	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AssignmentResponse{}, ErrAssignmentNotFound
		}

		return dto.AssignmentResponse{}, err
	}

	if payload.Title != nil {
		assignment.Title = *payload.Title
	}

	if payload.Description != nil {
		assignment.Description = *payload.Description
	}

	if payload.DueDate != nil {
		dueDate, err := time.Parse(time.RFC3339, *payload.DueDate)
		if err != nil {
			return dto.AssignmentResponse{}, fmt.Errorf("invalid due date: %w", err)
		}

		if !dueDate.After(s.now()) {
			return dto.AssignmentResponse{}, fmt.Errorf("due date must be in the future")
		}

		assignment.DueDate = dueDate
	}

	if file != nil {
		url, err := s.uploadFile(ctx, file)
		if err != nil {
			return dto.AssignmentResponse{}, err
		}
		assignment.FileURL = url
	}

	if err := s.repo.Update(ctx, &assignment); err != nil {
		return dto.AssignmentResponse{}, err
	}

	s.logger.Info().Uint("assignment_id", assignment.ID).Msg("assignment updated")

	return dto.NewAssignmentResponse(assignment), nil
}

func (s *assignmentService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAssignmentNotFound
		}
		return err
	}

	s.logger.Info().Uint("assignment_id", id).Msg("assignment deleted")
	return nil
}

func (s *assignmentService) uploadFile(ctx context.Context, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	url, err := s.uploader.Upload(ctx, file.Filename, src)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return url, nil
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func calculateTotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		if total == 0 {
			return 1
		}
		return 1
	}
	return int(math.Ceil(float64(total) / float64(pageSize)))
}
