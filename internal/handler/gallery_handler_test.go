package handler_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

type mockGalleryService struct {
	lastTags     []string
	lastSearch   string
	lastPage     int
	lastPageSize int
	response     dto.GalleryListResponse
	err          error
}

func (m *mockGalleryService) List(_ context.Context, tags []string, search string, page, pageSize int) (dto.GalleryListResponse, error) {
	m.lastTags = append([]string(nil), tags...)
	m.lastSearch = search
	m.lastPage = page
	m.lastPageSize = pageSize
	if m.err != nil {
		return dto.GalleryListResponse{}, m.err
	}
	return m.response, nil
}

func (m *mockGalleryService) Seed(_ context.Context, _ repository.GalleryFilter, _ func(context.Context) error) error {
	return nil
}

func TestGalleryHandler_ListSuccess(t *testing.T) {
	svc := &mockGalleryService{response: dto.GalleryListResponse{
		Items:      []dto.GalleryItemResponse{{ID: 1, Title: "Art Show"}},
		Pagination: dto.PaginationMeta{Page: 2, PageSize: 15, TotalItems: 1, TotalPages: 1},
	}}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewGalleryHandler(svc, logger).Register(app.Group("/api/gallery"))

	req := httptest.NewRequest(http.MethodGet, "/api/gallery?tags=%20art%20,%20robotics%20,%20&search=%20Show%20&page=2&pageSize=15", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body struct {
		Success bool                    `json:"success"`
		Data    dto.GalleryListResponse `json:"data"`
		Message string                  `json:"message"`
	}
	decodeResponse(t, resp, &body)

	require.True(t, body.Success)
	require.Equal(t, "gallery items retrieved", body.Message)
	require.Equal(t, []string{"art", "robotics"}, svc.lastTags)
	require.Equal(t, "Show", svc.lastSearch)
	require.Equal(t, 2, svc.lastPage)
	require.Equal(t, 15, svc.lastPageSize)
}

func TestGalleryHandler_InvalidPage(t *testing.T) {
	svc := &mockGalleryService{}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewGalleryHandler(svc, logger).Register(app.Group("/api/gallery"))

	req := httptest.NewRequest(http.MethodGet, "/api/gallery?page=bad", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGalleryHandler_ServiceError(t *testing.T) {
	svc := &mockGalleryService{err: errors.New("boom")}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewGalleryHandler(svc, logger).Register(app.Group("/api/gallery"))

	req := httptest.NewRequest(http.MethodGet, "/api/gallery", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
