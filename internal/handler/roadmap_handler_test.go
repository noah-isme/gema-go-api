package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/handler"
)

type stubRoadmapService struct {
	result dto.RoadmapStageListResult
	err    error
}

func (s stubRoadmapService) ListStages(context.Context, dto.RoadmapStageListRequest) (dto.RoadmapStageListResult, error) {
	return s.result, s.err
}

func TestRoadmapHandlerListStages(t *testing.T) {
	app := fiber.New()
	result := dto.RoadmapStageListResult{
		Items: []dto.RoadmapStageResponse{
			{
				ID:          1,
				Slug:        "intro",
				Title:       "Intro",
				Description: "Basics",
				UpdatedAt:   time.Now(),
			},
		},
		Pagination: dto.PaginationMeta{Page: 1, PageSize: 10, TotalItems: 1, TotalPages: 1},
		Filters:    dto.RoadmapStageFilters{Sort: "sequence"},
		CacheHit:   true,
	}

	handler := handler.NewRoadmapHandler(stubRoadmapService{result: result}, zerolog.Nop())
	handler.Register(app.Group("/api/roadmap"))

	req := httptest.NewRequest(http.MethodGet, "/api/roadmap/stages", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
