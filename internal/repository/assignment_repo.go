package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// AssignmentFilter describes pagination & search options.
type AssignmentFilter struct {
	Search   string
	Sort     string
	Page     int
	PageSize int
}

// AssignmentRepository defines persistence operations for assignments.
type AssignmentRepository interface {
	List(ctx context.Context) ([]models.Assignment, error)
	ListWithFilter(ctx context.Context, filter AssignmentFilter) ([]models.Assignment, int64, error)
	GetByID(ctx context.Context, id uint) (models.Assignment, error)
	Create(ctx context.Context, assignment *models.Assignment) error
	Update(ctx context.Context, assignment *models.Assignment) error
	Delete(ctx context.Context, id uint) error
}

type assignmentRepository struct {
	db *gorm.DB
}

// NewAssignmentRepository instantiates a GORM-backed repository.
func NewAssignmentRepository(db *gorm.DB) AssignmentRepository {
	return &assignmentRepository{db: db}
}

func (r *assignmentRepository) List(ctx context.Context) ([]models.Assignment, error) {
	var assignments []models.Assignment
	if err := r.db.WithContext(ctx).Order("due_date ASC").Find(&assignments).Error; err != nil {
		return nil, err
	}

	return assignments, nil
}

func (r *assignmentRepository) ListWithFilter(ctx context.Context, filter AssignmentFilter) ([]models.Assignment, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Assignment{})

	if filter.Search != "" {
		pattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := normalizeAssignmentSort(filter.Sort)
	if order != "" {
		query = query.Order(order)
	}

	if filter.PageSize > 0 {
		page := filter.Page
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	var assignments []models.Assignment
	if err := query.Find(&assignments).Error; err != nil {
		return nil, 0, err
	}

	return assignments, total, nil
}

func (r *assignmentRepository) GetByID(ctx context.Context, id uint) (models.Assignment, error) {
	var assignment models.Assignment
	if err := r.db.WithContext(ctx).First(&assignment, id).Error; err != nil {
		return models.Assignment{}, err
	}

	return assignment, nil
}

func (r *assignmentRepository) Create(ctx context.Context, assignment *models.Assignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

func (r *assignmentRepository) Update(ctx context.Context, assignment *models.Assignment) error {
	return r.db.WithContext(ctx).Save(assignment).Error
}

func (r *assignmentRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Assignment{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func normalizeAssignmentSort(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "due_date", "due_date:asc", "due_date.asc":
		return "due_date ASC"
	case "-due_date", "due_date:desc", "due_date.desc":
		return "due_date DESC"
	case "updated_at", "updated_at:asc", "updated_at.asc":
		return "updated_at ASC"
	case "-updated_at", "updated_at:desc", "updated_at.desc":
		return "updated_at DESC"
	case "title", "title:asc", "title.asc":
		return "title ASC"
	case "-title", "title:desc", "title.desc":
		return "title DESC"
	default:
		return "due_date ASC"
	}
}
