package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/service"
)

type stubStudentDashboardService struct {
	response dto.StudentDashboardResponse
	err      error
	calls    int
	lastID   uint
	cacheHit bool
}

func (s *stubStudentDashboardService) GetDashboard(_ context.Context, studentID uint) (dto.StudentDashboardResponse, bool, error) {
	s.calls++
	s.lastID = studentID
	if s.err != nil {
		return dto.StudentDashboardResponse{}, false, s.err
	}
	return s.response, s.cacheHit, nil
}

func TestStudentDashboardHandler_Success(t *testing.T) {
	now := time.Now()
	response := dto.StudentDashboardResponse{
		Summary: dto.ProgressSummary{TotalAssignments: 4, Graded: 2, CompletionRate: 50},
		Pending: []dto.AssignmentProgress{
			{AssignmentID: 7, Title: "Essay", DueDate: now},
		},
	}
	svc := &stubStudentDashboardService{response: response}
	logger := zerolog.Nop()

	app := fiber.New()
	group := app.Group("/api/v2/student", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(33))
		c.Locals("user_role", "student")
		return c.Next()
	})
	handler.NewStudentDashboardHandler(svc, logger).Register(group)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/student/dashboard", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var payload struct {
		Success bool                         `json:"success"`
		Message string                       `json:"message"`
		Data    dto.StudentDashboardResponse `json:"data"`
		Meta    map[string]interface{}       `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	resp.Body.Close()

	require.True(t, payload.Success)
	require.Equal(t, "dashboard retrieved", payload.Message)
	require.Equal(t, response.Summary.TotalAssignments, payload.Data.Summary.TotalAssignments)
	require.Equal(t, uint(33), svc.lastID)
	require.Equal(t, 1, svc.calls)
	require.NotNil(t, payload.Meta)
	require.Contains(t, payload.Meta, "cache_hit")
}

func TestStudentDashboardHandler_Unauthorized(t *testing.T) {
	svc := &stubStudentDashboardService{}
	logger := zerolog.Nop()

	app := fiber.New()
	handler.NewStudentDashboardHandler(svc, logger).Register(app.Group("/api/v2/student"))

	req := httptest.NewRequest(http.MethodGet, "/api/v2/student/dashboard", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var payload struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	resp.Body.Close()

	require.False(t, payload.Success)
	require.NotEmpty(t, payload.Message)
	require.Equal(t, 0, svc.calls)
}

var _ service.StudentDashboardService = (*stubStudentDashboardService)(nil)
