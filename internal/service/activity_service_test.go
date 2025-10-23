package service

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

type memoryActivityRepo struct {
	entries []models.ActivityLog
}

func (m *memoryActivityRepo) Create(ctx context.Context, entry *models.ActivityLog) error {
	entry.ID = uint(len(m.entries) + 1)
	entry.CreatedAt = time.Now()
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *memoryActivityRepo) List(ctx context.Context, filter repository.ActivityLogFilter) ([]models.ActivityLog, int64, error) {
	return append([]models.ActivityLog(nil), m.entries...), int64(len(m.entries)), nil
}

func TestActivityServiceRecordMasksEmail(t *testing.T) {
	repo := &memoryActivityRepo{}
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewActivityService(repo, validate, testLogger())

	entry, err := svc.Record(context.Background(), ActivityEntry{
		ActorID:    1,
		ActorRole:  "Admin",
		Action:     "student.updated",
		EntityType: "student",
		EntityID:   ptrUint(5),
		Metadata: map[string]interface{}{
			"email": "student@example.com",
			"field": "status",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "***", entry.Metadata["email"])
	require.Equal(t, "status", entry.Metadata["field"])
	require.Equal(t, uint(1), entry.ActorID)
}

func ptrUint(v uint) *uint {
	return &v
}
