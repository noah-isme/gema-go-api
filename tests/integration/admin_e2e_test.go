package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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
	"github.com/noah-isme/gema-go-api/internal/middleware"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
	"github.com/noah-isme/gema-go-api/internal/router"
	"github.com/noah-isme/gema-go-api/internal/service"
)

type integrationUploader struct{}

func (integrationUploader) Upload(_ context.Context, name string, _ io.Reader) (string, error) {
	return "https://files.test/" + name, nil
}

func setupAdminApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Student{}, &models.Assignment{}, &models.Submission{}, &models.SubmissionGradeHistory{}, &models.ActivityLog{}))

	validate := validator.New(validator.WithRequiredStructEnabled())
	logger := zerolog.New(io.Discard)

	assignmentRepo := repository.NewAssignmentRepository(db)
	submissionRepo := repository.NewSubmissionRepository(db)
	adminStudentRepo := repository.NewAdminStudentRepository(db)
	adminSubmissionRepo := repository.NewAdminSubmissionRepository(db)
	analyticsRepo := repository.NewAdminAnalyticsRepository(db)
	activityRepo := repository.NewActivityLogRepository(db)

	uploader := integrationUploader{}

	assignmentService := service.NewAssignmentService(assignmentRepo, validate, uploader, logger)
	submissionService := service.NewSubmissionService(submissionRepo, assignmentRepo, validate, uploader, logger)
	activityService := service.NewActivityService(activityRepo, validate, logger)
	adminStudentService := service.NewAdminStudentService(adminStudentRepo, validate, activityService, logger)
	adminAssignmentService := service.NewAdminAssignmentService(assignmentRepo, validate, activityService, logger)
	adminGradingService := service.NewAdminGradingService(adminSubmissionRepo, validate, activityService, logger)
	adminAnalyticsService := service.NewAdminAnalyticsService(analyticsRepo, nil, 0, logger)

	assignmentHandler := handler.NewAssignmentHandler(assignmentService, validate, logger)
	submissionHandler := handler.NewSubmissionHandler(submissionService, validate, logger)
	adminStudentHandler := handler.NewAdminStudentHandler(adminStudentService, logger)
	adminAssignmentHandler := handler.NewAdminAssignmentHandler(adminAssignmentService, logger)
	adminGradingHandler := handler.NewAdminGradingHandler(adminGradingService, logger)
	adminAnalyticsHandler := handler.NewAdminAnalyticsHandler(adminAnalyticsService, logger)
	adminActivityHandler := handler.NewAdminActivityHandler(activityService, logger)

	app := fiber.New()
	middleware.Register(app, middleware.Config{Logger: &logger})

	router.Register(app, config.Config{AppName: "Test", JWTSecret: "secret"}, router.Dependencies{
		AssignmentHandler:      assignmentHandler,
		SubmissionHandler:      submissionHandler,
		AdminStudentHandler:    adminStudentHandler,
		AdminAssignmentHandler: adminAssignmentHandler,
		AdminGradingHandler:    adminGradingHandler,
		AdminAnalyticsHandler:  adminAnalyticsHandler,
		AdminActivityHandler:   adminActivityHandler,
		JWTMiddleware: func(c *fiber.Ctx) error {
			if strings.HasPrefix(c.Path(), "/api/admin") {
				c.Locals("user_id", uint(9001))
				c.Locals("user_role", "admin")
			} else {
				c.Locals("user_id", uint(1))
				c.Locals("user_role", "student")
			}
			return c.Next()
		},
	})

	return app, db
}

func decode[T any](t *testing.T, resp *http.Response, target *T) {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, target))
}

func TestAdminEndToEndFlow(t *testing.T) {
	app, db := setupAdminApp(t)

	student := models.Student{Name: "Siti", Email: "siti@example.com", Status: models.StudentStatusInactive, Class: "XI-A"}
	require.NoError(t, db.Create(&student).Error)

	// Step 1: update student status via admin API
	status := "active"
	notes := "Verified via QA"
	payload := map[string]interface{}{
		"status": status,
		"notes":  notes,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/students/"+strconv.Itoa(int(student.ID)), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, res.StatusCode)

	var studentResp struct {
		Success bool                     `json:"success"`
		Data    dto.AdminStudentResponse `json:"data"`
		Message string                   `json:"message"`
	}
	decode(t, res, &studentResp)
	require.True(t, studentResp.Success)
	require.Equal(t, status, studentResp.Data.Status)
	require.Equal(t, notes, studentResp.Data.Notes)

	// Step 2: admin creates assignment
	dueDate := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	assignmentPayload := map[string]interface{}{
		"title":       "Project Plan",
		"description": "Submit milestone report",
		"due_date":    dueDate,
		"max_score":   100,
	}
	assignmentBody, err := json.Marshal(assignmentPayload)
	require.NoError(t, err)

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/assignments", bytes.NewReader(assignmentBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, createResp.StatusCode)

	var assignmentResp struct {
		Success bool                        `json:"success"`
		Data    dto.AdminAssignmentResponse `json:"data"`
	}
	decode(t, createResp, &assignmentResp)
	require.True(t, assignmentResp.Success)

	// Step 3: student uploads submission
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	require.NoError(t, writer.WriteField("assignment_id", strconv.Itoa(int(assignmentResp.Data.ID))))
	require.NoError(t, writer.WriteField("student_id", strconv.Itoa(int(student.ID))))
	file, err := writer.CreateFormFile("file", "submission.zip")
	require.NoError(t, err)
	_, err = file.Write([]byte("zip-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	submissionReq := httptest.NewRequest(http.MethodPost, "/api/v2/tutorial/submissions", buf)
	submissionReq.Header.Set("Content-Type", writer.FormDataContentType())
	submissionResp, err := app.Test(submissionReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, submissionResp.StatusCode)

	var submissionBody struct {
		Success bool                   `json:"success"`
		Data    dto.SubmissionResponse `json:"data"`
	}
	decode(t, submissionResp, &submissionBody)
	require.True(t, submissionBody.Success)
	require.Equal(t, assignmentResp.Data.ID, submissionBody.Data.Assignment.ID)

	// Step 4: admin grades submission
	gradePayload := map[string]interface{}{
		"score":    85,
		"feedback": "Great progress",
	}
	gradeBody, err := json.Marshal(gradePayload)
	require.NoError(t, err)

	gradeReq := httptest.NewRequest(http.MethodPatch, "/api/admin/submissions/"+strconv.Itoa(int(submissionBody.Data.ID))+"/grade", bytes.NewReader(gradeBody))
	gradeReq.Header.Set("Content-Type", "application/json")
	gradeResp, err := app.Test(gradeReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, gradeResp.StatusCode)

	var gradedBody struct {
		Success bool                   `json:"success"`
		Data    dto.SubmissionResponse `json:"data"`
	}
	decode(t, gradeResp, &gradedBody)
	require.True(t, gradedBody.Success)
	require.NotNil(t, gradedBody.Data.Grade)
	require.Equal(t, 85.0, *gradedBody.Data.Grade)

	// Step 5: admin fetches analytics
	analyticsReq := httptest.NewRequest(http.MethodGet, "/api/admin/analytics", nil)
	analyticsResp, err := app.Test(analyticsReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, analyticsResp.StatusCode)

	var analyticsBody struct {
		Success bool                       `json:"success"`
		Data    dto.AdminAnalyticsResponse `json:"data"`
	}
	decode(t, analyticsResp, &analyticsBody)
	require.True(t, analyticsBody.Success)
	require.Equal(t, int64(1), analyticsBody.Data.ActiveStudents)
	require.Equal(t, int64(1), analyticsBody.Data.OnTimeSubmissions)
	require.Equal(t, int64(0), analyticsBody.Data.LateSubmissions)
	require.Contains(t, analyticsBody.Data.GradeDistribution, "75-89")
	require.Equal(t, int64(1), analyticsBody.Data.GradeDistribution["75-89"])
}
