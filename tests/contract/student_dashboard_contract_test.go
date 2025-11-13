package contract_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/handler"
)

type stubDashboardService struct {
	response dto.StudentDashboardResponse
}

func (s stubDashboardService) GetDashboard(context.Context, uint) (dto.StudentDashboardResponse, bool, error) {
	return s.response, false, nil
}

func TestStudentDashboardContract(t *testing.T) {
	schemaPath, err := filepath.Abs(filepath.Join("..", "contracts", "student_dashboard.schema.json"))
	require.NoError(t, err)

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile("file://" + schemaPath)
	require.NoError(t, err)

	now := time.Now().UTC()
	response := dto.StudentDashboardResponse{
		Summary: dto.ProgressSummary{
			TotalAssignments: 5,
			Submitted:        3,
			Graded:           2,
			Pending:          2,
			Overdue:          1,
			AverageGrade:     88.5,
			CompletionRate:   40.0,
		},
		Pending: []dto.AssignmentProgress{
			{
				AssignmentID:  10,
				Title:         "Lab Report",
				DueDate:       now,
				FileURL:       "https://cdn.example.com/report.pdf",
				Status:        "submitted",
				SubmissionID:  ptrUint(99),
				SubmissionURL: "https://cdn.example.com/submission.zip",
				Grade:         ptrFloat(90),
				Feedback:      "Great job",
				UpdatedAt:     now,
				Overdue:       false,
			},
		},
		RecentSubmissions: []dto.SubmissionActivity{
			{
				SubmissionID:   55,
				AssignmentID:   10,
				AssignmentName: "Lab Report",
				Status:         "graded",
				Grade:          ptrFloat(90),
				Feedback:       "Solid work",
				CreatedAt:      now.Add(-48 * time.Hour),
				UpdatedAt:      now,
			},
		},
	}

	svc := stubDashboardService{response: response}
	handler := handler.NewStudentDashboardHandler(svc, zerolog.Nop())

	app := fiber.New()
	group := app.Group("/api/v2/student", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(1))
		c.Locals("user_role", "student")
		return c.Next()
	})
	handler.Register(group)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/student/dashboard", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	var payload interface{}
	require.NoError(t, json.Unmarshal(body, &payload))
	require.NoError(t, schema.Validate(payload))
}

func ptrUint(v uint) *uint {
	return &v
}

func ptrFloat(v float64) *float64 {
	return &v
}
