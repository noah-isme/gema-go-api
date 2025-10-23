package performance_test

import (
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
	"github.com/noah-isme/gema-go-api/internal/service"
)

func setupAnalyticsPerformanceApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Student{}, &models.Assignment{}, &models.Submission{}))

	// Seed dataset
	now := time.Now().UTC()
	assignments := []models.Assignment{
		{Title: "Module 1", DueDate: now.Add(12 * time.Hour), MaxScore: 100},
		{Title: "Module 2", DueDate: now.Add(24 * time.Hour), MaxScore: 100},
	}
	for _, assignment := range assignments {
		require.NoError(t, db.Create(&assignment).Error)
	}

	students := []models.Student{
		{Name: "Ani", Email: "ani@example.com", Status: models.StudentStatusActive},
		{Name: "Budi", Email: "budi@example.com", Status: models.StudentStatusActive},
		{Name: "Cici", Email: "cici@example.com", Status: models.StudentStatusActive},
	}
	for _, student := range students {
		require.NoError(t, db.Create(&student).Error)
	}

	grade := 88.0
	for idx, assignment := range assignments {
		for _, student := range students {
			submission := models.Submission{
				AssignmentID: assignment.ID,
				StudentID:    student.ID,
				FileURL:      "https://files.test/submission.zip",
				Status:       models.SubmissionStatusGraded,
				Grade:        &grade,
				CreatedAt:    now.Add(time.Duration(idx) * time.Hour),
				UpdatedAt:    now.Add(time.Duration(idx) * time.Hour),
			}
			require.NoError(t, db.Create(&submission).Error)
		}
	}

	analyticsRepo := repository.NewAdminAnalyticsRepository(db)
	analyticsService := service.NewAdminAnalyticsService(analyticsRepo, nil, 0, zerolog.Nop())
	analyticsHandler := handler.NewAdminAnalyticsHandler(analyticsService, zerolog.Nop())

	app := fiber.New()
	analyticsHandler.Register(app.Group("/api/admin/analytics"))

	return app, db
}

func TestAdminAnalyticsP95LatencyBelow250ms(t *testing.T) {
	app, db := setupAnalyticsPerformanceApp(t)
	t.Cleanup(func() { _ = db })

	runs := 40
	durations := make([]time.Duration, 0, runs)

	for i := 0; i < runs; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics", nil)
		start := time.Now()
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		durations = append(durations, time.Since(start))
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	index := int(math.Ceil(0.95*float64(len(durations)))) - 1
	if index < 0 {
		index = 0
	}
	p95 := durations[index]

	require.LessOrEqual(t, p95, 250*time.Millisecond)
}
