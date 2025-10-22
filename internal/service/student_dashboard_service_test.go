package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

func TestStudentDashboardServiceAggregationAndCaching(t *testing.T) {
	mini, err := miniredis.Run()
	require.NoError(t, err)
	defer mini.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Student{}, &models.Assignment{}, &models.Submission{}))

	studentID := uint(1)
	student := models.Student{ID: studentID, Name: "John Doe", Email: "john@example.com"}
	require.NoError(t, db.Create(&student).Error)

	now := time.Now().UTC()
	assignments := []models.Assignment{
		{Title: "Assignment 1", Description: "A1", DueDate: now.Add(48 * time.Hour)},
		{Title: "Assignment 2", Description: "A2", DueDate: now.Add(24 * time.Hour)},
		{Title: "Assignment 3", Description: "A3", DueDate: now.Add(-24 * time.Hour)},
	}
	for i := range assignments {
		require.NoError(t, db.Create(&assignments[i]).Error)
	}

	submissions := []models.Submission{
		{
			AssignmentID: assignments[0].ID,
			StudentID:    studentID,
			FileURL:      "https://example.com/sub1",
			Status:       models.SubmissionStatusSubmitted,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			AssignmentID: assignments[1].ID,
			StudentID:    studentID,
			FileURL:      "https://example.com/sub2",
			Status:       models.SubmissionStatusGraded,
			Grade:        floatPointer(90),
			Feedback:     "Great job",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	for i := range submissions {
		require.NoError(t, db.Create(&submissions[i]).Error)
	}

	assignmentRepo := repository.NewAssignmentRepository(db)
	submissionRepo := repository.NewSubmissionRepository(db)

	svc := NewStudentDashboardService(assignmentRepo, submissionRepo, redisClient, time.Minute, zerolog.Nop())

	ctx := context.Background()
	first, err := svc.GetDashboard(ctx, studentID)
	require.NoError(t, err)
	require.Equal(t, 3, first.Summary.TotalAssignments)
	require.Equal(t, 2, first.Summary.Submitted)
	require.Equal(t, 1, first.Summary.Graded)
	require.Equal(t, 2, first.Summary.Pending)
	require.Equal(t, 1, first.Summary.Overdue)
	require.InDelta(t, 90.0, first.Summary.AverageGrade, 0.01)
	require.InDelta(t, 33.33, first.Summary.CompletionRate, 0.5)
	require.Len(t, first.Pending, 2)
	require.Len(t, first.RecentSubmissions, 2)

	// Modify database to ensure cached response is returned unchanged.
	require.NoError(t, db.Model(&assignments[0]).Update("title", "Changed Title").Error)

	second, err := svc.GetDashboard(ctx, studentID)
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func floatPointer(v float64) *float64 {
	return &v
}

func TestStudentDashboardCacheHit(t *testing.T) {
	mini, err := miniredis.Run()
	require.NoError(t, err)
	defer mini.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Student{}, &models.Assignment{}, &models.Submission{}))

	assignmentRepo := repository.NewAssignmentRepository(db)
	submissionRepo := repository.NewSubmissionRepository(db)

	svc := NewStudentDashboardService(assignmentRepo, submissionRepo, redisClient, time.Minute, zerolog.Nop())

	studentID := uint(10)
	ctx := context.Background()

	// Seed cache manually
	cached := dto.StudentDashboardResponse{
		Summary: dto.ProgressSummary{TotalAssignments: 1},
	}
	payload, err := json.Marshal(cached)
	require.NoError(t, err)
	require.NoError(t, redisClient.Set(ctx, "dashboard:student:10", payload, time.Minute).Err())

	response, err := svc.GetDashboard(ctx, studentID)
	require.NoError(t, err)
	require.Equal(t, cached, response)
}
