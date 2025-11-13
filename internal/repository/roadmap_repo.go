package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// RoadmapStageFilter describes filters applied to roadmap queries.
type RoadmapStageFilter struct {
	Search   string
	Tags     []string
	Sort     string
	Page     int
	PageSize int
}

// RoadmapStageRepository exposes roadmap persistence helpers.
type RoadmapStageRepository interface {
	List(ctx context.Context, filter RoadmapStageFilter) ([]models.RoadmapStage, int64, error)
}

type roadmapStageRepository struct {
	db *gorm.DB
}

// NewRoadmapStageRepository constructs a repository.
func NewRoadmapStageRepository(db *gorm.DB) RoadmapStageRepository {
	return &roadmapStageRepository{db: db}
}

func (r *roadmapStageRepository) List(ctx context.Context, filter RoadmapStageFilter) ([]models.RoadmapStage, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.RoadmapStage{})

	if filter.Search != "" {
		pattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", pattern, pattern)
	}

	for _, tag := range filter.Tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		query = query.Where("tags LIKE ?", "%|"+tag+"|%")
	}

	order := roadmapSortClause(filter.Sort)
	if order != "" {
		query = query.Order(order)
	}

	countQuery := query.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.PageSize > 0 {
		page := filter.Page
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	var stages []models.RoadmapStage
	if err := query.Find(&stages).Error; err != nil {
		return nil, 0, err
	}
	return stages, total, nil
}

func roadmapSortClause(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "sequence", "sequence:asc", "sequence.asc":
		return "sequence ASC"
	case "-sequence", "sequence:desc", "sequence.desc":
		return "sequence DESC"
	case "recent", "updated_at", "updated_at:desc", "-updated_at":
		return "updated_at DESC"
	default:
		return "sequence ASC"
	}
}
