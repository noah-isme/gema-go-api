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

type stubAnalyticsService struct {
	response dto.AdminAnalyticsResponse
}

func (s stubAnalyticsService) GetSummary(context.Context) (dto.AdminAnalyticsResponse, error) {
	return s.response, nil
}

func TestAdminAnalyticsContract(t *testing.T) {
	schemaPath, err := filepath.Abs(filepath.Join("..", "contracts", "admin_analytics.schema.json"))
	require.NoError(t, err)

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile("file://" + schemaPath)
	require.NoError(t, err)

	analytics := dto.AdminAnalyticsResponse{
		ActiveStudents:    3,
		OnTimeSubmissions: 2,
		LateSubmissions:   1,
		GradeDistribution: dto.GradeDistributionResponse{"90-100": 1, "75-89": 2},
		WeeklyEngagement: []dto.WeeklyEngagementPoint{
			{WeekStart: time.Now().AddDate(0, 0, -7).UTC(), Submissions: 5},
		},
		GeneratedAt: time.Now().UTC(),
		CacheHit:    false,
	}

	serviceStub := stubAnalyticsService{response: analytics}
	handler := handler.NewAdminAnalyticsHandler(serviceStub, zerolog.Nop())

	app := fiber.New()
	handler.Register(app.Group("/api/admin/analytics"))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var payload interface{}
	require.NoError(t, json.Unmarshal(body, &payload))
	require.NoError(t, schema.Validate(payload))
}
