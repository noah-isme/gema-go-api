package middleware

import (
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rs/zerolog"
)

// Config customises the middleware registration pipeline.
type Config struct {
	Logger *zerolog.Logger
}

// Register attaches the common middlewares used across the API.
func Register(app *fiber.App, cfg Config) {
	requestLogger := zerolog.New(io.Discard)
	if cfg.Logger != nil {
		requestLogger = *cfg.Logger
	}

	app.Use(recover.New())
	app.Use(CorrelationID())
	app.Use(Observability(requestLogger))
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}))
}
