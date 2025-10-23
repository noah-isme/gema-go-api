package handler_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
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

type handlerWebLabUploader struct{}

func (u *handlerWebLabUploader) Upload(_ context.Context, name string, _ io.Reader) (string, error) {
	return "https://example.com/uploads/" + name, nil
}

func setupWebLabApp(t *testing.T) (*fiber.App, *gorm.DB, models.Student, models.WebAssignment) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Student{}, &models.WebAssignment{}, &models.WebSubmission{}))

	student := models.Student{Name: "Budi", Email: fmt.Sprintf("budi+%d@example.com", time.Now().UnixNano())}
	require.NoError(t, db.Create(&student).Error)

	assignment := models.WebAssignment{Title: "Hero Section", Requirements: "Bangun hero section", Rubric: "Struktur & aksesibilitas"}
	assignment.SetAssets([]string{"assets/hero.png"})
	require.NoError(t, db.Create(&assignment).Error)

	validate := validator.New(validator.WithRequiredStructEnabled())
	logger := zerolog.New(io.Discard)
	uploader := &handlerWebLabUploader{}

	webLabService := service.NewWebLabService(
		repository.NewWebAssignmentRepository(db),
		repository.NewWebSubmissionRepository(db),
		repository.NewStudentRepository(db),
		validate,
		uploader,
		logger,
	)

	app := fiber.New()
	webLabHandler := handler.NewWebLabHandler(webLabService, validate, logger)

	router.Register(app, config.Config{AppName: "Test", JWTSecret: "secret"}, router.Dependencies{
		WebLabHandler: webLabHandler,
		JWTMiddleware: func(c *fiber.Ctx) error {
			c.Locals("user_id", student.ID)
			return c.Next()
		},
	})

	return app, db, student, assignment
}

func TestWebLabHandler_ListAssignments(t *testing.T) {
	app, _, _, _ := setupWebLabApp(t)

	req := httptest.NewRequest("GET", "/api/v2/web-lab/assignments", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body struct {
		Success bool                        `json:"success"`
		Data    []dto.WebAssignmentResponse `json:"data"`
		Message string                      `json:"message"`
	}
	decodeResponse(t, resp, &body)

	require.True(t, body.Success)
	require.NotEmpty(t, body.Data)
	require.Equal(t, "assignments retrieved", body.Message)
}

func TestWebLabHandler_SubmissionUsesJWTStudent(t *testing.T) {
	app, db, student, assignment := setupWebLabApp(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("assignment_id", strconv.FormatUint(uint64(assignment.ID), 10)))
	require.NoError(t, writer.WriteField("student_id", "9999"))
	part, err := writer.CreateFormFile("file", "project.zip")
	require.NoError(t, err)
	_, err = part.Write(buildZip([]zipEntry{
		{Name: "index.html", Content: []byte("<html><head></head><body>Hi</body></html>")},
	}))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/api/v2/web-lab/submissions", body)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var bodyResp struct {
		Success bool                      `json:"success"`
		Data    dto.WebSubmissionResponse `json:"data"`
		Message string                    `json:"message"`
	}
	decodeResponse(t, resp, &bodyResp)

	require.True(t, bodyResp.Success)
	require.Equal(t, "submission processed", bodyResp.Message)
	require.Equal(t, assignment.ID, bodyResp.Data.AssignmentID)
	require.Equal(t, student.ID, bodyResp.Data.StudentID)
	require.NotNil(t, bodyResp.Data.Score)

	var stored models.WebSubmission
	require.NoError(t, db.First(&stored).Error)
	require.Equal(t, student.ID, stored.StudentID)
}

func TestWebLabHandler_SubmissionRejectsInvalidZip(t *testing.T) {
	app, _, _, assignment := setupWebLabApp(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("assignment_id", strconv.FormatUint(uint64(assignment.ID), 10)))
	part, err := writer.CreateFormFile("file", "project.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("not zip"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/api/v2/web-lab/submissions", body)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var bodyResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	decodeResponse(t, resp, &bodyResp)
	require.False(t, bodyResp.Success)
}

type zipEntry struct {
	Name    string
	Content []byte
}

func buildZip(entries []zipEntry) []byte {
	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)

	for _, entry := range entries {
		w, err := writer.Create(entry.Name)
		if err != nil {
			panic(err)
		}
		if len(entry.Content) > 0 {
			if _, err := w.Write(entry.Content); err != nil {
				panic(err)
			}
		}
	}

	if err := writer.Close(); err != nil {
		panic(err)
	}

	return buf.Bytes()
}
