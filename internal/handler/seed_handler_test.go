package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/service"
)

type mockSeedService struct {
	announcementsErr  error
	galleryErr        error
	lastToken         string
	lastAnnouncements []models.Announcement
	lastGallery       []models.GalleryItem
	affected          int64
}

func (m *mockSeedService) SeedAnnouncements(_ context.Context, token string, items []models.Announcement) (int64, error) {
	m.lastToken = token
	m.lastAnnouncements = items
	if m.announcementsErr != nil {
		return 0, m.announcementsErr
	}
	return m.affected, nil
}

func (m *mockSeedService) SeedGallery(_ context.Context, token string, items []models.GalleryItem) (int64, error) {
	m.lastToken = token
	m.lastGallery = items
	if m.galleryErr != nil {
		return 0, m.galleryErr
	}
	return m.affected, nil
}

func TestSeedHandler_AnnouncementsSuccess(t *testing.T) {
	svc := &mockSeedService{affected: 2}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewSeedHandler(svc, logger).Register(app.Group("/api/seed"))

	payload := map[string]interface{}{"items": []models.Announcement{{Slug: "welcome", Title: "Welcome"}}}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/seed/announcements", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Seed-Token", "secret")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Affected int64 `json:"affected"`
		} `json:"data"`
	}
	decodeResponse(t, resp, &response)

	require.True(t, response.Success)
	require.Equal(t, int64(2), response.Data.Affected)
	require.Equal(t, "secret", svc.lastToken)
	require.Len(t, svc.lastAnnouncements, 1)
}

func TestSeedHandler_GalleryErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		statusCode int
		message    string
	}{
		{name: "disabled", err: service.ErrSeedDisabled, statusCode: fiber.StatusForbidden, message: "seeding disabled"},
		{name: "unauthorized", err: service.ErrSeedUnauthorized, statusCode: fiber.StatusForbidden, message: "invalid token"},
		{name: "generic", err: errors.New("boom"), statusCode: fiber.StatusInternalServerError, message: "seed operation failed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockSeedService{galleryErr: tc.err}
			logger := zerolog.New(io.Discard)
			app := fiber.New()
			handler.NewSeedHandler(svc, logger).Register(app.Group("/api/seed"))

			payload := map[string]interface{}{"items": []models.GalleryItem{{Slug: "event", Title: "Event"}}}
			body, err := json.Marshal(payload)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/seed/gallery", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Seed-Token", "secret")

			resp, err := app.Test(req)
			require.NoError(t, err)
			require.Equal(t, tc.statusCode, resp.StatusCode)

			var response struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			decodeResponse(t, resp, &response)
			require.False(t, response.Success)
			require.Equal(t, tc.message, response.Message)
		})
	}
}

func TestSeedHandler_InvalidPayload(t *testing.T) {
	svc := &mockSeedService{}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewSeedHandler(svc, logger).Register(app.Group("/api/seed"))

	req := httptest.NewRequest(http.MethodPost, "/api/seed/announcements", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	require.Nil(t, svc.lastAnnouncements)
}
