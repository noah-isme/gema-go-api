package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
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

type submissionTestUploader struct{}

func (s *submissionTestUploader) Upload(_ context.Context, name string, _ io.Reader) (string, error) {
	return "https://files.test/" + name, nil
}

func setupSubmissionApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Student{}, &models.Assignment{}, &models.Submission{}))

	validate := validator.New(validator.WithRequiredStructEnabled())
	logger := zerolog.New(io.Discard)
	uploader := &submissionTestUploader{}

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

	return app, db
}

func TestSubmissionHandlerUploadAndGrade(t *testing.T) {
	app, db := setupSubmissionApp(t)

	student := models.Student{Name: "Jane", Email: "jane@example.com"}
	require.NoError(t, db.Create(&student).Error)

	assignment := models.Assignment{
		Title:       "Lab Report",
		Description: "Submit lab",
		DueDate:     time.Now().Add(3 * time.Hour),
	}
	require.NoError(t, db.Create(&assignment).Error)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("assignment_id", strconv.FormatUint(uint64(assignment.ID), 10)))
	require.NoError(t, writer.WriteField("student_id", strconv.FormatUint(uint64(student.ID), 10)))
	part, err := writer.CreateFormFile("file", "submission.zip")
	require.NoError(t, err)
	_, err = part.Write([]byte("zip"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/api/v2/tutorial/submissions", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var createResp struct {
		Success bool                   `json:"success"`
		Data    dto.SubmissionResponse `json:"data"`
		Message string                 `json:"message"`
	}
	decodeResponse(t, resp, &createResp)
	require.True(t, createResp.Success)
	require.Equal(t, "submission created", createResp.Message)
	require.NotZero(t, createResp.Data.ID)
	require.Equal(t, assignment.ID, createResp.Data.Assignment.ID)
	require.Equal(t, assignment.Title, createResp.Data.Assignment.Title)
	require.WithinDuration(t, assignment.DueDate, createResp.Data.Assignment.DueDate, time.Second)

	gradePayload := map[string]interface{}{
		"status":   "graded",
		"grade":    95,
		"feedback": "Excellent",
	}
	gradeBody, err := json.Marshal(gradePayload)
	require.NoError(t, err)

	updateReq := httptest.NewRequest("PATCH", "/api/v2/tutorial/submissions/"+strconv.FormatUint(uint64(createResp.Data.ID), 10), bytes.NewReader(gradeBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := app.Test(updateReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, updateResp.StatusCode)

	var updateBody struct {
		Success bool                   `json:"success"`
		Data    dto.SubmissionResponse `json:"data"`
		Message string                 `json:"message"`
	}
	decodeResponse(t, updateResp, &updateBody)
	require.True(t, updateBody.Success)
	require.Equal(t, "submission updated", updateBody.Message)
	require.NotNil(t, updateBody.Data.Grade)
	require.Equal(t, 95.0, *updateBody.Data.Grade)
	require.Equal(t, "graded", updateBody.Data.Status)
}
