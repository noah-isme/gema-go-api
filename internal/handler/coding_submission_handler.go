package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// CodingSubmissionHandler exposes submission endpoints for the coding lab.
type CodingSubmissionHandler struct {
	service   service.CodingSubmissionService
	validator *validator.Validate
	logger    zerolog.Logger
}

// NewCodingSubmissionHandler constructs the handler.
func NewCodingSubmissionHandler(service service.CodingSubmissionService, validator *validator.Validate, logger zerolog.Logger) *CodingSubmissionHandler {
	return &CodingSubmissionHandler{
		service:   service,
		validator: validator,
		logger:    logger.With().Str("component", "coding_submission_handler").Logger(),
	}
}

// Register wires the handler endpoints into the router group.
func (h *CodingSubmissionHandler) Register(router fiber.Router) {
	router.Post("", h.create)
	router.Get("/:id", h.get)
	router.Post("/:id/evaluate", h.evaluate)
}

func (h *CodingSubmissionHandler) create(c *fiber.Ctx) error {
	var payload dto.CodingSubmissionRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.validator.Struct(payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	studentID := userIDFromContext(c)
	if studentID == 0 {
		return utils.SendError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	response, err := h.service.Submit(c.Context(), studentID, payload)
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "submission created", response)
}

func (h *CodingSubmissionHandler) get(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	response, err := h.service.Get(c.Context(), id, userIDFromContext(c), userRoleFromContext(c))
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "submission retrieved", response)
}

func (h *CodingSubmissionHandler) evaluate(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	evaluatorID := userIDFromContext(c)
	role := userRoleFromContext(c)
	if evaluatorID == 0 || role == "" {
		return utils.SendError(c, fiber.StatusForbidden, "insufficient permissions")
	}

	evaluation, err := h.service.Evaluate(c.Context(), id, evaluatorID, role)
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "submission evaluated", evaluation)
}

func (h *CodingSubmissionHandler) handleError(c *fiber.Ctx, err error) error {
	var validationErrors validator.ValidationErrors
	switch {
	case errors.Is(err, service.ErrUnsupportedLanguage):
		return utils.SendError(c, fiber.StatusBadRequest, "language not supported")
	case errors.Is(err, service.ErrCodingTaskNotFound), errors.Is(err, service.ErrCodingSubmissionNotFound):
		return utils.SendError(c, fiber.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrCodingSubmissionForbidden):
		return utils.SendError(c, fiber.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrEvaluatorUnavailable):
		return utils.SendError(c, fiber.StatusServiceUnavailable, "evaluator unavailable")
	case errors.As(err, &validationErrors):
		return utils.SendError(c, fiber.StatusBadRequest, validationErrors.Error())
	default:
		h.logger.Error().Err(err).Msg("submission operation failed")
		return utils.SendError(c, fiber.StatusInternalServerError, "internal server error")
	}
}
