package router

import (
	"github.com/gofiber/fiber/v2"

	"github.com/noah-isme/gema-go-api/internal/config"
	"github.com/noah-isme/gema-go-api/internal/handler"
)

// Register wires the HTTP routes into the fiber application.
func Register(app *fiber.App, cfg config.Config) {
	api := app.Group("/api/v1", func(c *fiber.Ctx) error {
		c.Set("X-Application", cfg.AppName)
		return c.Next()
	})

	api.Get("/health", handler.HealthCheck(cfg))
}
