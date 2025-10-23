package handler_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/models"
)

type mockAnnouncementService struct {
	lastPage     int
	lastPageSize int
	response     dto.AnnouncementListResponse
	err          error
}

func (m *mockAnnouncementService) ListActive(_ context.Context, page, pageSize int) (dto.AnnouncementListResponse, error) {
	m.lastPage = page
	m.lastPageSize = pageSize
	if m.err != nil {
		return dto.AnnouncementListResponse{}, m.err
	}
	return m.response, nil
}

func (m *mockAnnouncementService) Seed(_ context.Context, _ []models.Announcement) (int64, error) {
	return 0, nil
}

func TestAnnouncementHandler_ListSuccess(t *testing.T) {
	svc := &mockAnnouncementService{response: dto.AnnouncementListResponse{
		Items:      []dto.AnnouncementResponse{{ID: 1, Title: "Update", Body: "<p>news</p>", StartsAt: time.Now()}},
		Pagination: dto.PaginationMeta{Page: 1, PageSize: 20, TotalItems: 1, TotalPages: 1},
		CacheHit:   true,
	}}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewAnnouncementHandler(svc, logger).Register(app.Group("/api/announcements"))

	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Equal(t, "true", resp.Header.Get("X-Cache-Hit"))

	var body struct {
		Success bool                         `json:"success"`
		Data    dto.AnnouncementListResponse `json:"data"`
		Message string                       `json:"message"`
	}
	decodeResponse(t, resp, &body)

	require.True(t, body.Success)
	require.Equal(t, "announcements retrieved", body.Message)
	require.Equal(t, svc.response.Pagination.TotalItems, body.Data.Pagination.TotalItems)
	require.True(t, body.Data.CacheHit)
	require.Equal(t, 1, svc.lastPage)
	require.Equal(t, 20, svc.lastPageSize)
}

func TestAnnouncementHandler_InvalidPage(t *testing.T) {
	svc := &mockAnnouncementService{}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewAnnouncementHandler(svc, logger).Register(app.Group("/api/announcements"))

	req := httptest.NewRequest(http.MethodGet, "/api/announcements?page=oops", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestAnnouncementHandler_ServiceError(t *testing.T) {
	svc := &mockAnnouncementService{err: errors.New("boom")}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewAnnouncementHandler(svc, logger).Register(app.Group("/api/announcements"))

	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
