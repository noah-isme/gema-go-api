package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

type stubActivityRecorder struct {
	entries []ActivityEntry
}

func (s *stubActivityRecorder) Record(_ context.Context, entry ActivityEntry) (dto.AdminActivityResponse, error) {
	s.entries = append(s.entries, entry)
	return dto.AdminActivityResponse{Action: entry.Action, EntityType: entry.EntityType, EntityID: entry.EntityID}, nil
}

func setupAdminAssignmentService(t *testing.T) (*gorm.DB, AdminAssignmentService, *stubActivityRecorder) {
	t.Helper()

	dsn := fmt.Sprintf("file:admin_assignment_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Assignment{}))

	repo := repository.NewAssignmentRepository(db)
	validate := validator.New(validator.WithRequiredStructEnabled())
	activity := &stubActivityRecorder{}
	logger := zerolog.Nop()

	service := NewAdminAssignmentService(repo, validate, activity, logger)
	if concrete, ok := service.(*adminAssignmentService); ok {
		concrete.now = func() time.Time { return time.Date(2024, time.January, 5, 10, 0, 0, 0, time.UTC) }
	}

	return db, service, activity
}

func TestAdminAssignmentServiceCreate(t *testing.T) {
	db, service, activity := setupAdminAssignmentService(t)

	payload := dto.AdminAssignmentCreateRequest{
		Title:       " Midterm  ",
		Description: "  Solve the problems ",
		DueDate:     time.Date(2024, time.January, 7, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		MaxScore:    120,
		FileURL:     "https://cdn.test/task.pdf",
		Rubric: map[string]float64{
			"logic":     34.449,
			"structure": 65.551,
		},
	}

	actor := ActivityActor{ID: 99, Role: "admin"}
	response, err := service.Create(context.Background(), payload, actor)
	require.NoError(t, err)
	require.Equal(t, "Midterm", response.Title)
	require.Equal(t, "Solve the problems", response.Description)
	require.Equal(t, "https://cdn.test/task.pdf", response.FileURL)
	require.Equal(t, 120.0, response.MaxScore)
	require.Len(t, response.Rubric, 2)
	require.Equal(t, 34.45, response.Rubric["logic"])
	require.Equal(t, 65.55, response.Rubric["structure"])

	var stored models.Assignment
	require.NoError(t, db.First(&stored, response.ID).Error)
	require.WithinDuration(t, time.Date(2024, time.January, 7, 10, 0, 0, 0, time.UTC), stored.DueDate, time.Second)

	require.Len(t, activity.entries, 1)
	require.Equal(t, "assignment.created", activity.entries[0].Action)
	require.Equal(t, actor.ID, activity.entries[0].ActorID)
	require.NotNil(t, activity.entries[0].EntityID)
	require.Equal(t, stored.ID, *activity.entries[0].EntityID)
}

func TestAdminAssignmentServiceCreateRejectsPastDueDate(t *testing.T) {
	_, service, _ := setupAdminAssignmentService(t)

	payload := dto.AdminAssignmentCreateRequest{
		Title:    "Past",
		DueDate:  time.Date(2023, time.December, 31, 23, 0, 0, 0, time.UTC).Format(time.RFC3339),
		MaxScore: 10,
	}

	_, err := service.Create(context.Background(), payload, ActivityActor{ID: 1, Role: "admin"})
	require.ErrorIs(t, err, ErrAdminAssignmentInvalidDueDate)
}

func TestAdminAssignmentServiceUpdate(t *testing.T) {
	_, service, activity := setupAdminAssignmentService(t)
	createPayload := dto.AdminAssignmentCreateRequest{
		Title:    "Quiz",
		DueDate:  time.Date(2024, time.January, 7, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		MaxScore: 50,
	}
	actor := ActivityActor{ID: 7, Role: "teacher"}
	created, err := service.Create(context.Background(), createPayload, actor)
	require.NoError(t, err)
	activity.entries = nil

	newTitle := "Quiz Updated"
	newDue := time.Date(2024, time.January, 10, 10, 0, 0, 0, time.UTC).Format(time.RFC3339)
	newScore := 60.0
	updatePayload := dto.AdminAssignmentUpdateRequest{
		Title:    &newTitle,
		DueDate:  &newDue,
		MaxScore: &newScore,
	}

	updated, err := service.Update(context.Background(), created.ID, updatePayload, actor)
	require.NoError(t, err)
	require.Equal(t, newTitle, updated.Title)
	require.Equal(t, newScore, updated.MaxScore)
	require.WithinDuration(t, time.Date(2024, time.January, 10, 10, 0, 0, 0, time.UTC), updated.DueDate, time.Second)

	require.Len(t, activity.entries, 1)
	require.Equal(t, "assignment.updated", activity.entries[0].Action)
	require.Contains(t, activity.entries[0].Metadata["fields"], "title")
	require.Contains(t, activity.entries[0].Metadata["fields"], "due_date")
	require.Contains(t, activity.entries[0].Metadata["fields"], "max_score")
}

func TestAdminAssignmentServiceDelete(t *testing.T) {
	db, service, activity := setupAdminAssignmentService(t)
	created, err := service.Create(context.Background(), dto.AdminAssignmentCreateRequest{
		Title:    "Delete Me",
		DueDate:  time.Date(2024, time.January, 7, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		MaxScore: 20,
	}, ActivityActor{ID: 11, Role: "admin"})
	require.NoError(t, err)
	activity.entries = nil

	err = service.Delete(context.Background(), created.ID, ActivityActor{ID: 11, Role: "admin"})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.Assignment{}).Where("id = ?", created.ID).Count(&count).Error)
	require.Zero(t, count)
	require.Len(t, activity.entries, 1)
	require.Equal(t, "assignment.deleted", activity.entries[0].Action)

	err = service.Delete(context.Background(), created.ID, ActivityActor{ID: 11, Role: "admin"})
	require.ErrorIs(t, err, ErrAdminAssignmentNotFound)
}

func TestAdminAssignmentServiceGet(t *testing.T) {
	_, service, _ := setupAdminAssignmentService(t)
	actor := ActivityActor{ID: 1, Role: "admin"}
	created, err := service.Create(context.Background(), dto.AdminAssignmentCreateRequest{
		Title:    "Homework",
		DueDate:  time.Date(2024, time.January, 7, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		MaxScore: 30,
	}, actor)
	require.NoError(t, err)

	fetched, err := service.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
	require.Equal(t, created.Title, fetched.Title)

	_, err = service.Get(context.Background(), created.ID+100)
	require.ErrorIs(t, err, ErrAdminAssignmentNotFound)
}
