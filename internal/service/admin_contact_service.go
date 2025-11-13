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

// ErrAdminContactNotFound indicates submission missing.
var ErrAdminContactNotFound = errors.New("contact submission not found")

// AdminContactService exposes admin contact read operations.
type AdminContactService interface {
	List(ctx context.Context, req dto.AdminContactListRequest) (dto.AdminContactListResponse, error)
	Get(ctx context.Context, id uint) (dto.AdminContactResponse, error)
}

type adminContactService struct {
	repo   repository.ContactRepository
	logger zerolog.Logger
}

// NewAdminContactService constructs the contact admin service.
func NewAdminContactService(repo repository.ContactRepository, logger zerolog.Logger) AdminContactService {
	return &adminContactService{
		repo:   repo,
		logger: logger.With().Str("component", "admin_contact_service").Logger(),
	}
}

func (s *adminContactService) List(ctx context.Context, req dto.AdminContactListRequest) (dto.AdminContactListResponse, error) {
	filter := repository.AdminContactFilter{
		Search:   strings.TrimSpace(req.Search),
		Status:   strings.TrimSpace(req.Status),
		Sort:     strings.TrimSpace(req.Sort),
		Page:     normalizePage(req.Page),
		PageSize: clampPageSize(req.PageSize),
	}
	if filter.Sort == "" {
		filter.Sort = "created_at DESC"
	}

	submissions, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return dto.AdminContactListResponse{}, err
	}

	items := make([]dto.AdminContactResponse, 0, len(submissions))
	for _, submission := range submissions {
		items = append(items, toAdminContactResponse(submission))
	}

	pagination := dto.PaginationMeta{
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalItems: total,
		TotalPages: calculateTotalPages(total, filter.PageSize),
	}

	return dto.AdminContactListResponse{
		Items:      items,
		Pagination: pagination,
	}, nil
}

func (s *adminContactService) Get(ctx context.Context, id uint) (dto.AdminContactResponse, error) {
	submission, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.AdminContactResponse{}, ErrAdminContactNotFound
		}
		return dto.AdminContactResponse{}, err
	}
	return toAdminContactResponse(submission), nil
}

func toAdminContactResponse(model models.ContactSubmission) dto.AdminContactResponse {
	return dto.AdminContactResponse{
		ID:          model.ID,
		ReferenceID: model.ReferenceID,
		Name:        model.Name,
		Email:       maskEmailAddress(model.Email),
		Message:     model.Message,
		Status:      model.Status,
		Source:      model.Source,
		CreatedAt:   model.CreatedAt,
		DeliveredAt: model.DeliveredAt,
	}
}
