package router

import (
	"github.com/gofiber/fiber/v2"

	"github.com/noah-isme/gema-go-api/internal/config"
	"github.com/noah-isme/gema-go-api/internal/handler"
)

// Dependencies groups router dependencies for registration.
type Dependencies struct {
	AssignmentHandler       *handler.AssignmentHandler
	SubmissionHandler       *handler.SubmissionHandler
	StudentDashboardHandler *handler.StudentDashboardHandler
	JWTMiddleware           fiber.Handler
}

// Register wires the HTTP routes into the fiber application.
func Register(app *fiber.App, cfg config.Config, deps Dependencies) {
	api := app.Group("/api/v1", func(c *fiber.Ctx) error {
		c.Set("X-Application", cfg.AppName)
		return c.Next()
	})

	api.Get("/health", handler.HealthCheck(cfg))

	jwtMiddleware := deps.JWTMiddleware
	if jwtMiddleware == nil {
		jwtMiddleware = func(c *fiber.Ctx) error { return c.Next() }
	}

	if deps.AssignmentHandler != nil {
		tutorial := app.Group("/api/v2/tutorial", jwtMiddleware)
		assignmentGroup := tutorial.Group("/assignments")
		deps.AssignmentHandler.Register(assignmentGroup)

		if deps.SubmissionHandler != nil {
			submissionGroup := tutorial.Group("/submissions")
			deps.SubmissionHandler.Register(submissionGroup)
		}
	}

	if deps.StudentDashboardHandler != nil {
		student := app.Group("/api/v2/student", jwtMiddleware)
		deps.StudentDashboardHandler.Register(student)
	}
}
