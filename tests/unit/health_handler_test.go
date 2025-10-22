package unit

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/noah-isme/gema-go-api/internal/config"
	"github.com/noah-isme/gema-go-api/internal/handler"
)

type response struct {
	Success bool                   `json:"success"`
	Data    handler.HealthResponse `json:"data"`
}

func TestHealthCheck(t *testing.T) {
	cfg := config.Config{
		AppName: "GEMA API",
		AppEnv:  "test",
	}

	app := fiber.New()
	app.Get("/api/v1/health", handler.HealthCheck(cfg))

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var payload response
	err = json.NewDecoder(resp.Body).Decode(&payload)
	assert.NoError(t, err)
	assert.True(t, payload.Success)
	assert.Equal(t, "ok", payload.Data.Status)
	assert.Equal(t, cfg.AppName, payload.Data.Service)
	assert.Equal(t, cfg.AppEnv, payload.Data.Environment)
	assert.WithinDuration(t, time.Now().UTC(), payload.Data.Timestamp, 2*time.Second)
}
