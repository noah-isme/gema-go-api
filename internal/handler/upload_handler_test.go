package handler_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
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

type mockUploadService struct {
	lastUserID *uint
	response   dto.UploadResponse
	err        error
}

func (m *mockUploadService) Upload(_ context.Context, file *multipart.FileHeader, userID *uint) (dto.UploadResponse, error) {
	if file != nil {
		if _, err := file.Open(); err != nil {
			return dto.UploadResponse{}, err
		}
	}
	m.lastUserID = userID
	if m.err != nil {
		return dto.UploadResponse{}, m.err
	}
	return m.response, nil
}

func TestUploadHandler_Success(t *testing.T) {
	svc := &mockUploadService{response: dto.UploadResponse{URL: "https://cdn.example.com/file.png", SizeBytes: 123, MimeType: "image", Checksum: "abc", FileName: "file.png"}}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	group := app.Group("/api/upload", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(7))
		return c.Next()
	})
	handler.NewUploadHandler(svc, logger).Register(group)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "photo.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("png"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response struct {
		Success bool               `json:"success"`
		Data    dto.UploadResponse `json:"data"`
		Message string             `json:"message"`
	}
	decodeResponse(t, resp, &response)

	require.True(t, response.Success)
	require.Equal(t, "upload successful", response.Message)
	require.NotNil(t, svc.lastUserID)
	require.Equal(t, uint(7), *svc.lastUserID)
	require.Equal(t, svc.response.URL, response.Data.URL)
}

func TestUploadHandler_MissingFile(t *testing.T) {
	svc := &mockUploadService{}
	logger := zerolog.New(io.Discard)
	app := fiber.New()
	handler.NewUploadHandler(svc, logger).Register(app.Group("/api/upload"))

	req := httptest.NewRequest(http.MethodPost, "/api/upload", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUploadHandler_ServiceErrors(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		statusCode int
	}{
		{name: "too_large", err: service.ErrUploadTooLarge, statusCode: fiber.StatusRequestEntityTooLarge},
		{name: "type", err: service.ErrUploadTypeNotAllowed, statusCode: fiber.StatusBadRequest},
		{name: "scan", err: service.ErrUploadScanFailed, statusCode: fiber.StatusBadRequest},
		{name: "generic", err: errors.New("boom"), statusCode: fiber.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockUploadService{err: tc.err}
			logger := zerolog.New(io.Discard)
			app := fiber.New()
			handler.NewUploadHandler(svc, logger).Register(app.Group("/api/upload"))

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", "doc.pdf")
			require.NoError(t, err)
			_, err = part.Write([]byte("pdf"))
			require.NoError(t, err)
			require.NoError(t, writer.Close())

			req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := app.Test(req)
			require.NoError(t, err)
			require.Equal(t, tc.statusCode, resp.StatusCode)
		})
	}
}
