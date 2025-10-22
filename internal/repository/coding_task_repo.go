package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// CodingTaskQuery defines filters and pagination for coding tasks.
type CodingTaskQuery struct {
	Language   string
	Difficulty string
	Tags       []string
	Search     string
	Offset     int
	Limit      int
}

// CodingTaskRepository exposes persistence operations for coding tasks.
type CodingTaskRepository interface {
	List(ctx context.Context, query CodingTaskQuery) ([]models.CodingTask, int64, error)
	GetByID(ctx context.Context, id uint) (models.CodingTask, error)
}

// NewCodingTaskRepository constructs a coding task repository.
func NewCodingTaskRepository(db *gorm.DB) CodingTaskRepository {
	return &codingTaskRepository{db: db}
}

type codingTaskRepository struct {
	db *gorm.DB
}

func (r *codingTaskRepository) List(ctx context.Context, query CodingTaskQuery) ([]models.CodingTask, int64, error) {
	db := r.db.WithContext(ctx).Model(&models.CodingTask{})

	if query.Language != "" {
		db = db.Where("LOWER(language) = ?", strings.ToLower(query.Language))
	}

	if query.Difficulty != "" {
		db = db.Where("LOWER(difficulty) = ?", strings.ToLower(query.Difficulty))
	}

	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", strings.ToLower(query.Search))
		db = db.Where("LOWER(title) LIKE ? OR LOWER(prompt) LIKE ?", pattern, pattern)
	}

	if len(query.Tags) > 0 {
		for _, tag := range query.Tags {
			trimmed := strings.TrimSpace(tag)
			if trimmed == "" {
				continue
			}
			like := fmt.Sprintf("%%%s%%", strings.ToLower(trimmed))
			db = db.Where("LOWER(tags) LIKE ?", like)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}

	db = db.Order("created_at DESC")

	var tasks []models.CodingTask
	if err := db.Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func (r *codingTaskRepository) GetByID(ctx context.Context, id uint) (models.CodingTask, error) {
	var task models.CodingTask
	if err := r.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return models.CodingTask{}, err
	}
	return task, nil
}
