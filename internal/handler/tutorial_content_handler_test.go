package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
	"github.com/noah-isme/gema-go-api/internal/service"
)

func TestTutorialContentHandler_CreateAndList(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.TutorialArticle{}, &models.TutorialProject{}))

	validate := validator.New(validator.WithRequiredStructEnabled())
	articleRepo := repository.NewTutorialArticleRepository(db)
	projectRepo := repository.NewTutorialProjectRepository(db)
	contentService := service.NewTutorialContentService(articleRepo, projectRepo, validate, zerolog.Nop())
	contentHandler := handler.NewTutorialContentHandler(contentService, zerolog.Nop())

	app := fiber.New()
	contentHandler.RegisterPublic(app.Group("/api/tutorial"))
	contentHandler.RegisterAdmin(app.Group("/api/tutorial"))

	body := map[string]interface{}{
		"title":           "Concurrency in Go",
		"summary":         "Learn goroutines",
		"content":         "detailed content...",
		"tags":            []string{"go", "backend"},
		"reading_minutes": 8,
	}
	payload, _ := json.Marshal(body)

	createReq := httptest.NewRequest(http.MethodPost, "/api/tutorial/articles", bytes.NewReader(payload))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var createBody struct {
		Data dto.TutorialArticleResponse `json:"data"`
	}
	decodeResponse(t, createResp, &createBody)
	require.True(t, strings.HasPrefix(createBody.Data.Slug, "concurrency-in-go"))

	listReq := httptest.NewRequest(http.MethodGet, "/api/tutorial/articles?search=go&page=1&pageSize=5", nil)
	listResp, err := app.Test(listReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listResp.StatusCode)

	var listPayload struct {
		Data []dto.TutorialArticleResponse `json:"data"`
		Meta struct {
			Pagination dto.PaginationMeta         `json:"pagination"`
			Filters    dto.TutorialContentFilters `json:"filters"`
		} `json:"meta"`
	}
	decodeResponse(t, listResp, &listPayload)
	require.NotEmpty(t, listPayload.Data)
	require.Equal(t, 1, listPayload.Meta.Pagination.Page)
	require.Equal(t, "go", listPayload.Meta.Filters.Search)
}

func decodeResponse(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, json.Unmarshal(body, target))
}
