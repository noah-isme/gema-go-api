package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/config"
	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
	"github.com/noah-isme/gema-go-api/internal/router"
	"github.com/noah-isme/gema-go-api/internal/service"
)

type testAssignmentUploader struct{}

func (t *testAssignmentUploader) Upload(_ context.Context, name string, _ io.Reader) (string, error) {
	return "https://example.com/" + name, nil
}

func setupAssignmentApp(t *testing.T) *fiber.App {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Student{}, &models.Assignment{}, &models.Submission{}))

	validate := validator.New(validator.WithRequiredStructEnabled())
	logger := zerolog.New(io.Discard)
	uploader := &testAssignmentUploader{}

	assignmentRepo := repository.NewAssignmentRepository(db)
	submissionRepo := repository.NewSubmissionRepository(db)

	assignmentService := service.NewAssignmentService(assignmentRepo, validate, uploader, logger)
	submissionService := service.NewSubmissionService(submissionRepo, assignmentRepo, validate, uploader, logger)

	app := fiber.New()

	assignmentHandler := handler.NewAssignmentHandler(assignmentService, validate, logger)
	submissionHandler := handler.NewSubmissionHandler(submissionService, validate, logger)

	router.Register(app, config.Config{AppName: "Test", JWTSecret: "secret"}, router.Dependencies{
		AssignmentHandler: assignmentHandler,
		SubmissionHandler: submissionHandler,
		JWTMiddleware: func(c *fiber.Ctx) error {
			c.Locals("user_id", uint(1))
			return c.Next()
		},
	})

	return app
}

func TestAssignmentHandlerCreateAndList(t *testing.T) {
	app := setupAssignmentApp(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("title", "Data Structures"))
	require.NoError(t, writer.WriteField("description", "Implement heaps"))
	require.NoError(t, writer.WriteField("due_date", time.Now().Add(2*time.Hour).Format(time.RFC3339)))
	part, err := writer.CreateFormFile("file", "instructions.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("pdf"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/api/v2/tutorial/assignments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var createResp struct {
		Success bool                   `json:"success"`
		Data    dto.AssignmentResponse `json:"data"`
		Message string                 `json:"message"`
	}
	decodeResponse(t, resp, &createResp)
	require.True(t, createResp.Success)
	require.Equal(t, "assignment created", createResp.Message)
	require.NotZero(t, createResp.Data.ID)

	listReq := httptest.NewRequest("GET", "/api/v2/tutorial/assignments", nil)
	listResp, err := app.Test(listReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, listResp.StatusCode)

	var listBody struct {
		Success bool                     `json:"success"`
		Data    []dto.AssignmentResponse `json:"data"`
		Message string                   `json:"message"`
	}
	decodeResponse(t, listResp, &listBody)
	require.True(t, listBody.Success)
	require.NotEmpty(t, listBody.Data)
}

func decodeResponse(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, json.Unmarshal(data, target))
}
