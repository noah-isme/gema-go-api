package handler

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// StudentDashboardHandler exposes the student dashboard endpoint.
type StudentDashboardHandler struct {
	service service.StudentDashboardService
	logger  zerolog.Logger
}

// NewStudentDashboardHandler creates a new handler instance.
func NewStudentDashboardHandler(service service.StudentDashboardService, logger zerolog.Logger) *StudentDashboardHandler {
	return &StudentDashboardHandler{
		service: service,
		logger:  logger.With().Str("component", "student_dashboard_handler").Logger(),
	}
}

// Register attaches the dashboard endpoint.
func (h *StudentDashboardHandler) Register(router fiber.Router) {
	router.Get("/dashboard", h.getDashboard)
}

func (h *StudentDashboardHandler) getDashboard(c *fiber.Ctx) error {
	studentID, err := extractUserID(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, err.Error())
	}

	dashboard, err := h.service.GetDashboard(c.Context(), studentID)
	if err != nil {
		h.logger.Error().Err(err).Uint("student_id", studentID).Msg("failed to load dashboard")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to load dashboard")
	}

	return utils.SendSuccess(c, "dashboard retrieved", dashboard)
}

func extractUserID(c *fiber.Ctx) (uint, error) {
	value := c.Locals("user_id")
	if value == nil {
		return 0, fmt.Errorf("missing user context")
	}

	switch v := value.(type) {
	case uint:
		return v, nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("invalid user context")
		}
		return uint(v), nil
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid user context")
		}
		return uint(parsed), nil
	default:
		return 0, fmt.Errorf("invalid user context")
	}
}
