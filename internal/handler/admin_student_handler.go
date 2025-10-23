package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AdminStudentHandler wires admin student endpoints.
type AdminStudentHandler struct {
	service service.AdminStudentService
	logger  zerolog.Logger
}

// NewAdminStudentHandler constructs the handler.
func NewAdminStudentHandler(service service.AdminStudentService, logger zerolog.Logger) *AdminStudentHandler {
	return &AdminStudentHandler{
		service: service,
		logger:  logger.With().Str("component", "admin_student_handler").Logger(),
	}
}

// Register attaches student admin routes to the router group.
func (h *AdminStudentHandler) Register(router fiber.Router) {
	router.Get("", h.list)
	router.Get("/:id", h.get)
	router.Patch("/:id", h.update)
	router.Delete("/:id", h.delete)
}

func (h *AdminStudentHandler) list(c *fiber.Ctx) error {
	page, err := parseQueryInt(c, "page")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid page")
	}
	if page <= 0 {
		page = 1
	}

	pageSize, err := parseQueryInt(c, "page_size")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid page size")
	}
	if pageSize <= 0 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}

	req := dto.AdminStudentListRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   c.Query("search"),
		Class:    c.Query("class"),
		Status:   c.Query("status"),
		Sort:     c.Query("sort"),
	}

	response, err := h.service.List(c.Context(), req)
	if err != nil {
		requestLogger(h.logger, c).Error().Err(err).Msg("failed to list students")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to list students")
	}

	return utils.SendSuccess(c, "students retrieved", response)
}

func (h *AdminStudentHandler) get(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid identifier")
	}

	student, err := h.service.Get(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAdminStudentNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "student not found")
		}
		requestLogger(h.logger, c).Error().Err(err).Msg("failed to fetch student")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to fetch student")
	}

	return utils.SendSuccess(c, "student retrieved", student)
}

func (h *AdminStudentHandler) update(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid identifier")
	}

	var payload dto.AdminStudentUpdateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	actor := activityActorFromContext(c)
	student, err := h.service.Update(c.Context(), id, payload, actor)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAdminStudentNotFound):
			return utils.SendError(c, fiber.StatusNotFound, "student not found")
		case isValidationError(err):
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		default:
			requestLogger(h.logger, c).Error().Err(err).Msg("failed to update student")
			return utils.SendError(c, fiber.StatusInternalServerError, "failed to update student")
		}
	}

	return utils.SendSuccess(c, "student updated", student)
}

func (h *AdminStudentHandler) delete(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid identifier")
	}

	actor := activityActorFromContext(c)
	if err := h.service.Delete(c.Context(), id, actor); err != nil {
		if errors.Is(err, service.ErrAdminStudentNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "student not found")
		}
		requestLogger(h.logger, c).Error().Err(err).Msg("failed to delete student")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to delete student")
	}

	return utils.SendSuccess(c, "student deleted", fiber.Map{"id": id})
}
