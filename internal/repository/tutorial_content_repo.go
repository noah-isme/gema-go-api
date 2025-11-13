package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// TutorialContentFilter narrows tutorial content queries.
type TutorialContentFilter struct {
	Search   string
	Tags     []string
	Sort     string
	Page     int
	PageSize int
}

// TutorialArticleRepository persists tutorial articles.
type TutorialArticleRepository interface {
	List(ctx context.Context, filter TutorialContentFilter) ([]models.TutorialArticle, int64, error)
	GetByID(ctx context.Context, id uint) (models.TutorialArticle, error)
	Create(ctx context.Context, article *models.TutorialArticle) error
}

// TutorialProjectRepository persists tutorial projects.
type TutorialProjectRepository interface {
	List(ctx context.Context, filter TutorialContentFilter) ([]models.TutorialProject, int64, error)
	GetByID(ctx context.Context, id uint) (models.TutorialProject, error)
	Create(ctx context.Context, project *models.TutorialProject) error
}

type tutorialArticleRepository struct {
	db *gorm.DB
}

type tutorialProjectRepository struct {
	db *gorm.DB
}

// NewTutorialArticleRepository constructs an article repository.
func NewTutorialArticleRepository(db *gorm.DB) TutorialArticleRepository {
	return &tutorialArticleRepository{db: db}
}

// NewTutorialProjectRepository constructs a project repository.
func NewTutorialProjectRepository(db *gorm.DB) TutorialProjectRepository {
	return &tutorialProjectRepository{db: db}
}

func (r *tutorialArticleRepository) List(ctx context.Context, filter TutorialContentFilter) ([]models.TutorialArticle, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.TutorialArticle{})
	query = applyTutorialFilters(query, filter)
	order := tutorialSortClause(filter.Sort)
	if order != "" {
		query = query.Order(order)
	}
	return paginateTutorialArticles(query, filter.Page, filter.PageSize)
}

func (r *tutorialProjectRepository) List(ctx context.Context, filter TutorialContentFilter) ([]models.TutorialProject, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.TutorialProject{})
	query = applyTutorialFilters(query, filter)
	order := tutorialSortClause(filter.Sort)
	if order != "" {
		query = query.Order(order)
	}
	return paginateTutorialProjects(query, filter.Page, filter.PageSize)
}

func (r *tutorialArticleRepository) GetByID(ctx context.Context, id uint) (models.TutorialArticle, error) {
	var article models.TutorialArticle
	err := r.db.WithContext(ctx).First(&article, id).Error
	return article, err
}

func (r *tutorialProjectRepository) GetByID(ctx context.Context, id uint) (models.TutorialProject, error) {
	var project models.TutorialProject
	err := r.db.WithContext(ctx).First(&project, id).Error
	return project, err
}

func (r *tutorialArticleRepository) Create(ctx context.Context, article *models.TutorialArticle) error {
	return r.db.WithContext(ctx).Create(article).Error
}

func (r *tutorialProjectRepository) Create(ctx context.Context, project *models.TutorialProject) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func applyTutorialFilters(query *gorm.DB, filter TutorialContentFilter) *gorm.DB {
	if filter.Search != "" {
		pattern := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(summary) LIKE ?", pattern, pattern)
	}

	for _, tag := range filter.Tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		query = query.Where("tags LIKE ?", "%|"+tag+"|%")
	}

	return query
}

func paginateTutorialArticles(query *gorm.DB, page, pageSize int) ([]models.TutorialArticle, int64, error) {
	countQuery := query.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, page, pageSize)

	var records []models.TutorialArticle
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func paginateTutorialProjects(query *gorm.DB, page, pageSize int) ([]models.TutorialProject, int64, error) {
	countQuery := query.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, page, pageSize)

	var records []models.TutorialProject
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func applyPagination(query *gorm.DB, page, pageSize int) *gorm.DB {
	if pageSize <= 0 {
		return query
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize
	return query.Offset(offset).Limit(pageSize)
}

func tutorialSortClause(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "recent", "-updated_at", "updated_at:desc", "updated_at.desc":
		return "updated_at DESC"
	case "oldest", "updated_at", "updated_at:asc", "updated_at.asc":
		return "updated_at ASC"
	case "title", "title:asc", "title.asc":
		return "title ASC"
	case "-title", "title:desc", "title.desc":
		return "title DESC"
	default:
		return "updated_at DESC"
	}
}
