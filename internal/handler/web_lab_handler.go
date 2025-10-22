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

// WebLabHandler exposes web lab assignment and submission endpoints.
type WebLabHandler struct {
	service   service.WebLabService
	validator *validator.Validate
	logger    zerolog.Logger
}

// NewWebLabHandler builds a web lab handler instance.
func NewWebLabHandler(service service.WebLabService, validator *validator.Validate, logger zerolog.Logger) *WebLabHandler {
	return &WebLabHandler{
		service:   service,
		validator: validator,
		logger:    logger.With().Str("component", "web_lab_handler").Logger(),
	}
}

// Register wires the routes below /api/v2/web-lab.
func (h *WebLabHandler) Register(router fiber.Router) {
	assignments := router.Group("/assignments")
	assignments.Get("", h.listAssignments)
	assignments.Get("/:id", h.getAssignment)

	router.Post("/submissions", h.createSubmission)
}

func (h *WebLabHandler) listAssignments(c *fiber.Ctx) error {
	assignments, err := h.service.ListAssignments(c.Context())
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "assignments retrieved", assignments)
}

func (h *WebLabHandler) getAssignment(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	assignment, err := h.service.GetAssignment(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "assignment retrieved", assignment)
}

func (h *WebLabHandler) createSubmission(c *fiber.Ctx) error {
	studentID, err := studentIDFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusForbidden, err.Error())
	}

	assignmentID, err := parseFormUint(c, "assignment_id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	file, err := c.FormFile("file")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "file is required")
	}

	if assignmentID == nil {
		return utils.SendError(c, fiber.StatusBadRequest, "missing assignment_id")
	}

	payload := dto.WebSubmissionCreateRequest{
		AssignmentID: *assignmentID,
		StudentID:    studentID,
	}

	submission, err := h.service.CreateSubmission(c.Context(), payload, file)
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "submission processed", submission)
}

func (h *WebLabHandler) handleError(c *fiber.Ctx, err error) error {
	var validationErrors validator.ValidationErrors
	switch {
	case errors.Is(err, service.ErrWebAssignmentNotFound):
		return utils.SendError(c, fiber.StatusNotFound, "assignment not found")
	case errors.Is(err, service.ErrStudentNotFound):
		return utils.SendError(c, fiber.StatusForbidden, "student not found")
	case errors.Is(err, service.ErrWebSubmissionFileRequired):
		return utils.SendError(c, fiber.StatusBadRequest, "file is required")
	case errors.Is(err, service.ErrWebSubmissionUnsupportedType):
		return utils.SendError(c, fiber.StatusBadRequest, "submission must be a zip archive")
	case errors.Is(err, service.ErrWebSubmissionTooLarge):
		return utils.SendError(c, fiber.StatusRequestEntityTooLarge, "submission exceeds the 10 MB limit")
	case errors.Is(err, service.ErrWebSubmissionInvalidArchive):
		return utils.SendError(c, fiber.StatusBadRequest, "invalid zip archive")
	case errors.Is(err, service.ErrWebSubmissionDangerousFile):
		return utils.SendError(c, fiber.StatusBadRequest, "submission contains disallowed files")
	case errors.As(err, &validationErrors):
		return utils.SendError(c, fiber.StatusBadRequest, validationErrors.Error())
	default:
		h.logger.Error().Err(err).Msg("internal server error")
		return utils.SendError(c, fiber.StatusInternalServerError, "internal server error")
	}
}

func studentIDFromContext(c *fiber.Ctx) (uint, error) {
	value := c.Locals("user_id")
	if value == nil {
		return 0, errors.New("missing authenticated student")
	}

	switch v := value.(type) {
	case uint:
		return v, nil
	case int:
		if v < 0 {
			return 0, errors.New("invalid student identifier")
		}
		return uint(v), nil
	case int64:
		if v < 0 {
			return 0, errors.New("invalid student identifier")
		}
		return uint(v), nil
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, errors.New("invalid student identifier")
		}
		return uint(parsed), nil
	default:
		return 0, errors.New("invalid student identifier")
	}
}
