package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AdminAssignmentHandler wires admin assignment endpoints.
type AdminAssignmentHandler struct {
	service service.AdminAssignmentService
	logger  zerolog.Logger
}

// NewAdminAssignmentHandler constructs the handler.
func NewAdminAssignmentHandler(service service.AdminAssignmentService, logger zerolog.Logger) *AdminAssignmentHandler {
	return &AdminAssignmentHandler{
		service: service,
		logger:  logger.With().Str("component", "admin_assignment_handler").Logger(),
	}
}

// Register attaches assignment admin routes to the router group.
func (h *AdminAssignmentHandler) Register(router fiber.Router) {
	router.Post("", h.create)
	router.Patch("/:id", h.update)
	router.Delete("/:id", h.delete)
	router.Get("/:id", h.get)
}

func (h *AdminAssignmentHandler) create(c *fiber.Ctx) error {
	var payload dto.AdminAssignmentCreateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	actor := activityActorFromContext(c)
	assignment, err := h.service.Create(c.Context(), payload, actor)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAdminAssignmentInvalidDueDate):
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		case isValidationError(err):
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		default:
			requestLogger(h.logger, c).Error().Err(err).Msg("failed to create assignment")
			return utils.SendError(c, fiber.StatusInternalServerError, "failed to create assignment")
		}
	}

	return utils.SendSuccessWithStatus(c, fiber.StatusCreated, "assignment created", assignment)
}

func (h *AdminAssignmentHandler) update(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid identifier")
	}

	var payload dto.AdminAssignmentUpdateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	actor := activityActorFromContext(c)
	assignment, err := h.service.Update(c.Context(), id, payload, actor)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAdminAssignmentNotFound):
			return utils.SendError(c, fiber.StatusNotFound, "assignment not found")
		case errors.Is(err, service.ErrAdminAssignmentInvalidDueDate):
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		case isValidationError(err):
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		default:
			requestLogger(h.logger, c).Error().Err(err).Msg("failed to update assignment")
			return utils.SendError(c, fiber.StatusInternalServerError, "failed to update assignment")
		}
	}

	return utils.SendSuccess(c, "assignment updated", assignment)
}

func (h *AdminAssignmentHandler) delete(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid identifier")
	}

	actor := activityActorFromContext(c)
	if err := h.service.Delete(c.Context(), id, actor); err != nil {
		if errors.Is(err, service.ErrAdminAssignmentNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "assignment not found")
		}
		requestLogger(h.logger, c).Error().Err(err).Msg("failed to delete assignment")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to delete assignment")
	}

	return utils.SendSuccess(c, "assignment deleted", fiber.Map{"id": id})
}

func (h *AdminAssignmentHandler) get(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid identifier")
	}

	assignment, err := h.service.Get(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAdminAssignmentNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "assignment not found")
		}
		requestLogger(h.logger, c).Error().Err(err).Msg("failed to fetch assignment")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to fetch assignment")
	}

	return utils.SendSuccess(c, "assignment retrieved", assignment)
}
