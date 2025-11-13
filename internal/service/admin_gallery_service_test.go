package service

import (
	"context"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

func TestAdminGalleryServiceCreateAndList(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.GalleryItem{}))

	repo := repository.NewGalleryRepository(db)
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewAdminGalleryService(repo, validate, nil, zerolog.Nop())

	request := dto.AdminGalleryRequest{
		Title:    "Showcase",
		Caption:  "Demo",
		ImageURL: "https://cdn.example.com/img.png",
		Tags:     []string{"featured"},
	}

	item, err := svc.Create(context.Background(), request, ActivityActor{})
	require.NoError(t, err)
	require.Equal(t, "Showcase", item.Title)

	list, err := svc.List(context.Background(), dto.AdminGalleryListRequest{PageSize: 10})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
}
