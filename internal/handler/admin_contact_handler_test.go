package handler_test

import (
	"net/http"
	"net/http/httptest"
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

func TestAdminContactHandler_List(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.ContactSubmission{}))

	item := models.ContactSubmission{
		ReferenceID: "abc",
		Name:        "Rina",
		Email:       "rina@example.com",
		Message:     "Need help",
		Status:      "queued",
		CreatedAt:   time.Now(),
	}
	require.NoError(t, db.Create(&item).Error)

	repo := repository.NewContactRepository(db)
	svc := service.NewAdminContactService(repo, zerolog.Nop())
	h := handler.NewAdminContactHandler(svc, zerolog.Nop())

	app := fiber.New()
	h.Register(app.Group("/api/admin/contacts"))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/contacts?page=1&pageSize=5", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
