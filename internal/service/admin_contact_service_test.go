package service

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

func TestAdminContactServiceListMasksEmail(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.ContactSubmission{}))

	submissions := []models.ContactSubmission{
		{
			ReferenceID: "ref-1",
			Name:        "Rudi",
			Email:       "rudi@example.com",
			Message:     "Hello",
			Status:      "queued",
			CreatedAt:   time.Now(),
		},
	}
	for _, submission := range submissions {
		require.NoError(t, db.Create(&submission).Error)
	}

	repo := repository.NewContactRepository(db)
	svc := NewAdminContactService(repo, zerolog.Nop())

	result, err := svc.List(context.Background(), dto.AdminContactListRequest{PageSize: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "r***i@example.com", result.Items[0].Email)
}
