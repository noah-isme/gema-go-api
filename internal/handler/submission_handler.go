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

// SubmissionHandler manages submission endpoints.
type SubmissionHandler struct {
	service   service.SubmissionService
	validator *validator.Validate
	logger    zerolog.Logger
}

// NewSubmissionHandler builds a submission handler instance.
func NewSubmissionHandler(service service.SubmissionService, validator *validator.Validate, logger zerolog.Logger) *SubmissionHandler {
	return &SubmissionHandler{
		service:   service,
		validator: validator,
		logger:    logger.With().Str("component", "submission_handler").Logger(),
	}
}

// Register attaches the routes to the provided router group.
func (h *SubmissionHandler) Register(router fiber.Router) {
	router.Get("", h.list)
	router.Post("", h.create)
	router.Patch("/:id", h.update)
}

func (h *SubmissionHandler) list(c *fiber.Ctx) error {
	filter := dto.SubmissionFilter{}
	if assignmentID, err := parseQueryUint(c, "assignment_id"); err == nil && assignmentID != nil {
		filter.AssignmentID = assignmentID
	}
	if studentID, err := parseQueryUint(c, "student_id"); err == nil && studentID != nil {
		filter.StudentID = studentID
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	submissions, err := h.service.List(c.Context(), filter)
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "submissions retrieved", submissions)
}

func (h *SubmissionHandler) create(c *fiber.Ctx) error {
	var payload dto.SubmissionCreateRequest
	assignmentID, err := parseFormUint(c, "assignment_id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}
	studentID, err := parseFormUint(c, "student_id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	payload.AssignmentID = *assignmentID
	payload.StudentID = *studentID

	file, err := c.FormFile("file")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "file is required")
	}

	submission, err := h.service.Create(c.Context(), payload, file)
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "submission created", submission)
}

func (h *SubmissionHandler) update(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var payload dto.SubmissionUpdateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	submission, err := h.service.Update(c.Context(), id, payload)
	if err != nil {
		return h.handleError(c, err)
	}

	return utils.SendSuccess(c, "submission updated", submission)
}

func (h *SubmissionHandler) handleError(c *fiber.Ctx, err error) error {
	var validationErrors validator.ValidationErrors
	switch {
	case errors.Is(err, service.ErrAssignmentNotFound):
		return utils.SendError(c, fiber.StatusNotFound, "assignment not found")
	case errors.Is(err, service.ErrSubmissionNotFound):
		return utils.SendError(c, fiber.StatusNotFound, "submission not found")
	case errors.As(err, &validationErrors):
		return utils.SendError(c, fiber.StatusBadRequest, validationErrors.Error())
	default:
		h.logger.Error().Err(err).Msg("internal server error")
		return utils.SendError(c, fiber.StatusInternalServerError, "internal server error")
	}
}

func parseQueryUint(c *fiber.Ctx, key string) (*uint, error) {
	value := c.Query(key)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, err
	}
	result := uint(parsed)
	return &result, nil
}

func parseFormUint(c *fiber.Ctx, key string) (*uint, error) {
	value := c.FormValue(key)
	if value == "" {
		return nil, errors.New("missing " + key)
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, errors.New("invalid " + key)
	}
	result := uint(parsed)
	return &result, nil
}
