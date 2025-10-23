package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AdminGradingHandler wires grading endpoints for admins and teachers.
type AdminGradingHandler struct {
	service service.AdminGradingService
	logger  zerolog.Logger
}

// NewAdminGradingHandler constructs the handler.
func NewAdminGradingHandler(service service.AdminGradingService, logger zerolog.Logger) *AdminGradingHandler {
	return &AdminGradingHandler{
		service: service,
		logger:  logger.With().Str("component", "admin_grading_handler").Logger(),
	}
}

// Register attaches grading endpoints to the router group.
func (h *AdminGradingHandler) Register(router fiber.Router) {
	router.Patch("/:id/grade", h.grade)
}

func (h *AdminGradingHandler) grade(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid identifier")
	}

	var payload dto.AdminGradeSubmissionRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	actor := activityActorFromContext(c)
	submission, err := h.service.Grade(c.Context(), id, payload, actor)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAdminSubmissionNotFound):
			return utils.SendError(c, fiber.StatusNotFound, "submission not found")
		case errors.Is(err, service.ErrScoreExceedsMax):
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		case isValidationError(err):
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		default:
			h.logger.Error().Err(err).Uint("submission_id", id).Msg("failed to grade submission")
			return utils.SendError(c, fiber.StatusInternalServerError, "failed to grade submission")
		}
	}

	return utils.SendSuccess(c, "submission graded", submission)
}
