package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AssignmentHandler wires assignment HTTP routes.
type AssignmentHandler struct {
	service   service.AssignmentService
	validator *validator.Validate
	logger    zerolog.Logger
}

// NewAssignmentHandler constructs the handler.
func NewAssignmentHandler(service service.AssignmentService, validator *validator.Validate, logger zerolog.Logger) *AssignmentHandler {
	return &AssignmentHandler{
		service:   service,
		validator: validator,
		logger:    logger.With().Str("component", "assignment_handler").Logger(),
	}
}

// Register attaches assignment endpoints to the router group.
func (h *AssignmentHandler) Register(router fiber.Router) {
	router.Get("", h.list)
	router.Get("/:id", h.get)
	router.Post("", h.create)
	router.Patch("/:id", h.update)
	router.Delete("/:id", h.delete)
}

func (h *AssignmentHandler) list(c *fiber.Ctx) error {
	ctx := c.Context()
	assignments, err := h.service.List(ctx)
	if err != nil {
		return h.internalError(c, err)
	}

	return utils.SendSuccess(c, "assignments retrieved", assignments)
}

func (h *AssignmentHandler) get(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	assignment, err := h.service.Get(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAssignmentNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "assignment not found")
		}
		return h.internalError(c, err)
	}

	return utils.SendSuccess(c, "assignment retrieved", assignment)
}

func (h *AssignmentHandler) create(c *fiber.Ctx) error {
	payload := dto.AssignmentCreateRequest{
		Title:       c.FormValue("title"),
		Description: c.FormValue("description"),
		DueDate:     c.FormValue("due_date"),
	}

	file, err := c.FormFile("file")
	if err != nil {
		file = nil
	}

	assignment, err := h.service.Create(c.Context(), payload, file)
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "assignment created", assignment)
}

func (h *AssignmentHandler) update(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	payload := dto.AssignmentUpdateRequest{}
	if title := c.FormValue("title"); title != "" {
		payload.Title = &title
	}
	if description := c.FormValue("description"); description != "" {
		payload.Description = &description
	}
	if due := c.FormValue("due_date"); due != "" {
		payload.DueDate = &due
	}

	file, err := c.FormFile("file")
	if err != nil {
		file = nil
	}

	assignment, err := h.service.Update(c.Context(), id, payload, file)
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "assignment updated", assignment)
}

func (h *AssignmentHandler) delete(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		if errors.Is(err, service.ErrAssignmentNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "assignment not found")
		}
		return h.internalError(c, err)
	}

	return utils.SendSuccess(c, "assignment deleted", fiber.Map{"id": id})
}

func (h *AssignmentHandler) handleError(c *fiber.Ctx, err error) error {
	var validationErrors validator.ValidationErrors
	switch {
	case errors.Is(err, service.ErrAssignmentNotFound):
		return utils.SendError(c, fiber.StatusNotFound, "assignment not found")
	case errors.As(err, &validationErrors):
		return utils.SendError(c, fiber.StatusBadRequest, validationErrors.Error())
	default:
		return h.internalError(c, err)
	}
}

func (h *AssignmentHandler) internalError(c *fiber.Ctx, err error) error {
	h.logger.Error().Err(err).Msg("internal server error")
	return utils.SendError(c, fiber.StatusInternalServerError, "internal server error")
}

func parseUintParam(c *fiber.Ctx, name string) (uint, error) {
	value := c.Params(name)
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, errors.New("invalid identifier")
	}
	return uint(parsed), nil
}
