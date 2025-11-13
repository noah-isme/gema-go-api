package dto

import (
	"strings"
	"time"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// TutorialArticleResponse serializes tutorial articles for API responses.
type TutorialArticleResponse struct {
	ID             uint       `json:"id"`
	Slug           string     `json:"slug"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	Content        string     `json:"content"`
	Tags           []string   `json:"tags"`
	ThumbnailURL   string     `json:"thumbnail_url"`
	Author         string     `json:"author"`
	ReadingMinutes int        `json:"reading_minutes"`
	Status         string     `json:"status"`
	UpdatedAt      time.Time  `json:"updated_at"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
}

// TutorialProjectResponse serializes tutorial projects for API responses.
type TutorialProjectResponse struct {
	ID             uint      `json:"id"`
	Slug           string    `json:"slug"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	Content        string    `json:"content"`
	Difficulty     string    `json:"difficulty"`
	EstimatedHours int       `json:"estimated_hours"`
	Tags           []string  `json:"tags"`
	RepoURL        string    `json:"repo_url"`
	PreviewURL     string    `json:"preview_url"`
	Status         string    `json:"status"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TutorialArticleCreateRequest validates create requests.
type TutorialArticleCreateRequest struct {
	Title          string   `json:"title" validate:"required,min=3"`
	Summary        string   `json:"summary" validate:"omitempty,max=600"`
	Content        string   `json:"content" validate:"required,min=20"`
	Tags           []string `json:"tags" validate:"omitempty,dive,required"`
	ThumbnailURL   string   `json:"thumbnail_url" validate:"omitempty,url"`
	Author         string   `json:"author" validate:"omitempty,max=160"`
	ReadingMinutes int      `json:"reading_minutes" validate:"omitempty,gte=1,lte=300"`
	Status         string   `json:"status" validate:"omitempty,oneof=draft published archived"`
}

// TutorialProjectCreateRequest validates project creation payloads.
type TutorialProjectCreateRequest struct {
	Title          string   `json:"title" validate:"required,min=3"`
	Summary        string   `json:"summary" validate:"omitempty,max=600"`
	Content        string   `json:"content" validate:"required,min=20"`
	Difficulty     string   `json:"difficulty" validate:"omitempty,oneof=beginner intermediate advanced"`
	EstimatedHours int      `json:"estimated_hours" validate:"omitempty,gte=1,lte=200"`
	Tags           []string `json:"tags" validate:"omitempty,dive,required"`
	RepoURL        string   `json:"repo_url" validate:"omitempty,url"`
	PreviewURL     string   `json:"preview_url" validate:"omitempty,url"`
	Status         string   `json:"status" validate:"omitempty,oneof=draft published archived"`
}

// TutorialArticleListResult wraps article list data.
type TutorialArticleListResult struct {
	Items      []TutorialArticleResponse `json:"items"`
	Pagination PaginationMeta            `json:"pagination"`
	Filters    TutorialContentFilters    `json:"filters"`
}

// TutorialProjectListResult wraps project list data.
type TutorialProjectListResult struct {
	Items      []TutorialProjectResponse `json:"items"`
	Pagination PaginationMeta            `json:"pagination"`
	Filters    TutorialContentFilters    `json:"filters"`
}

// TutorialContentFilters captures applied filters.
type TutorialContentFilters struct {
	Tags   []string `json:"tags,omitempty"`
	Search string   `json:"search,omitempty"`
	Sort   string   `json:"sort,omitempty"`
}

// TutorialContentListRequest captures query params for list endpoints.
type TutorialContentListRequest struct {
	Page     int
	PageSize int
	Sort     string
	Search   string
	Tags     []string
}

// NewTutorialArticleResponse converts model -> DTO.
func NewTutorialArticleResponse(model models.TutorialArticle) TutorialArticleResponse {
	return TutorialArticleResponse{
		ID:             model.ID,
		Slug:           model.Slug,
		Title:          model.Title,
		Summary:        model.Summary,
		Content:        model.Content,
		Tags:           append([]string(nil), model.Tags...),
		ThumbnailURL:   model.ThumbnailURL,
		Author:         model.Author,
		ReadingMinutes: model.ReadingMinutes,
		Status:         strings.ToLower(model.Status),
		UpdatedAt:      model.UpdatedAt,
		PublishedAt:    model.PublishedAt,
	}
}

// NewTutorialProjectResponse converts model -> DTO.
func NewTutorialProjectResponse(model models.TutorialProject) TutorialProjectResponse {
	return TutorialProjectResponse{
		ID:             model.ID,
		Slug:           model.Slug,
		Title:          model.Title,
		Summary:        model.Summary,
		Content:        model.Content,
		Difficulty:     strings.ToLower(model.Difficulty),
		EstimatedHours: model.EstimatedHours,
		Tags:           append([]string(nil), model.Tags...),
		RepoURL:        model.RepoURL,
		PreviewURL:     model.PreviewURL,
		Status:         strings.ToLower(model.Status),
		UpdatedAt:      model.UpdatedAt,
	}
}
