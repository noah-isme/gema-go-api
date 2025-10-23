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

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/service"
)

type mockContactService struct {
	lastPayload dto.ContactRequest
	response    dto.ContactResponse
	err         error
}

func (m *mockContactService) Submit(_ context.Context, req dto.ContactRequest) (dto.ContactResponse, error) {
	m.lastPayload = req
	if m.err != nil {
		return dto.ContactResponse{}, m.err
	}
	return m.response, nil
}

func TestContactHandler_SubmitSuccess(t *testing.T) {
	svc := &mockContactService{response: dto.ContactResponse{ReferenceID: "ref-1", Status: "sent"}}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	group := app.Group("/api/contact", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(42))
		return c.Next()
	})
	handler.NewContactHandler(svc, logger).Register(group)

	payload := dto.ContactRequest{Name: "Alice", Email: "alice@example.com", Message: "Hello there!", Source: "web"}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "203.0.113.9")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response struct {
		Success bool                `json:"success"`
		Data    dto.ContactResponse `json:"data"`
		Message string              `json:"message"`
	}
	decodeResponse(t, resp, &response)

	require.True(t, response.Success)
	require.Equal(t, "contact submission accepted", response.Message)
	require.Equal(t, svc.response.ReferenceID, response.Data.ReferenceID)
	require.NotNil(t, svc.lastPayload.UserID)
	require.Equal(t, uint(42), *svc.lastPayload.UserID)
	require.NotEmpty(t, svc.lastPayload.IPAddress)
}

func TestContactHandler_HoneypotRejected(t *testing.T) {
	svc := &mockContactService{}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewContactHandler(svc, logger).Register(app.Group("/api/contact"))

	payload := map[string]string{"name": "Bob", "email": "bob@example.com", "message": "Hi", "_note": "spam"}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	require.Empty(t, svc.lastPayload.Name)
}

func TestContactHandler_ServiceErrors(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		statusCode int
	}{
		{name: "spam", err: service.ErrContactSpam, statusCode: fiber.StatusBadRequest},
		{name: "duplicate", err: service.ErrContactDuplicate, statusCode: fiber.StatusTooManyRequests},
		{name: "generic", err: errors.New("boom"), statusCode: fiber.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockContactService{err: tc.err}
			logger := zerolog.New(io.Discard)
			app := fiber.New()
			handler.NewContactHandler(svc, logger).Register(app.Group("/api/contact"))

			payload := dto.ContactRequest{Name: "Alice", Email: "alice@example.com", Message: "Hello there!"}
			body, err := json.Marshal(payload)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			require.Equal(t, tc.statusCode, resp.StatusCode)
		})
	}
}
