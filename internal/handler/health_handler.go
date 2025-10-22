package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/noah-isme/gema-go-api/internal/config"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// HealthResponse represents the payload returned by the health endpoint.
type HealthResponse struct {
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
	Service     string    `json:"service"`
	Environment string    `json:"environment"`
}

// HealthCheck returns a handler that reports application health information.
func HealthCheck(cfg config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		payload := HealthResponse{
			Status:      "ok",
			Timestamp:   time.Now().UTC(),
			Service:     cfg.AppName,
			Environment: cfg.AppEnv,
		}

		return utils.SendSuccess(c, "service healthy", payload)
	}
}
