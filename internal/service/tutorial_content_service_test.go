package service

import (
	"context"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

func TestTutorialContentServiceCreateAndListArticles(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.TutorialArticle{}, &models.TutorialProject{}))

	validate := validator.New(validator.WithRequiredStructEnabled())
	articleRepo := repository.NewTutorialArticleRepository(db)
	projectRepo := repository.NewTutorialProjectRepository(db)

	svc := NewTutorialContentService(articleRepo, projectRepo, validate, testLogger())

	payload := dto.TutorialArticleCreateRequest{
		Title:          "Deep Learning Basics",
		Summary:        "Intro to DL",
		Content:        "This is the full content for DL basics.",
		Tags:           []string{"AI", "ML"},
		ReadingMinutes: 12,
		Status:         "published",
	}

	created, err := svc.CreateArticle(context.Background(), payload)
	require.NoError(t, err)
	require.NotZero(t, created.ID)
	require.Contains(t, created.Slug, "deep-learning-basics")
	require.Equal(t, "published", created.Status)

	list, err := svc.ListArticles(context.Background(), dto.TutorialContentListRequest{
		Search: "deep",
		Tags:   []string{"ai"},
		Page:   1,
	})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, created.ID, list.Items[0].ID)
	require.Equal(t, int64(1), list.Pagination.TotalItems)
	require.Equal(t, []string{"ai"}, list.Filters.Tags)
}

func TestTutorialContentServiceCreateProject(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.TutorialArticle{}, &models.TutorialProject{}))

	validate := validator.New(validator.WithRequiredStructEnabled())
	articleRepo := repository.NewTutorialArticleRepository(db)
	projectRepo := repository.NewTutorialProjectRepository(db)

	svc := NewTutorialContentService(articleRepo, projectRepo, validate, testLogger())
	payload := dto.TutorialProjectCreateRequest{
		Title:          "Weather App",
		Summary:        "Build a weather dashboard",
		Content:        "Project instructions ...",
		Difficulty:     "intermediate",
		EstimatedHours: 6,
		Tags:           []string{"frontend"},
		RepoURL:        "https://github.com/example/weather",
		PreviewURL:     "https://cdn.example.com/weather.png",
		Status:         "draft",
	}

	project, err := svc.CreateProject(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, "intermediate", project.Difficulty)

	list, err := svc.ListProjects(context.Background(), dto.TutorialContentListRequest{
		PageSize: 1,
	})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.True(t, strings.HasPrefix(list.Items[0].Slug, "weather-app"))
	require.Equal(t, int64(1), list.Pagination.TotalItems)
}
