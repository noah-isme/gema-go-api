package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// UploadHandler handles generic file uploads.
type UploadHandler struct {
	service service.UploadService
	logger  zerolog.Logger
}

// NewUploadHandler constructs an upload handler.
func NewUploadHandler(service service.UploadService, logger zerolog.Logger) *UploadHandler {
	return &UploadHandler{
		service: service,
		logger:  logger.With().Str("component", "upload_handler").Logger(),
	}
}

// Register wires upload routes.
func (h *UploadHandler) Register(router fiber.Router) {
	router.Post("", h.upload)
}

func (h *UploadHandler) upload(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "file is required")
	}

	var userID *uint
	if id, ok := c.Locals("user_id").(uint); ok && id > 0 {
		userID = &id
	}

	result, err := h.service.Upload(c.Context(), file, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUploadTooLarge):
			return utils.SendError(c, fiber.StatusRequestEntityTooLarge, err.Error())
		case errors.Is(err, service.ErrUploadTypeNotAllowed), errors.Is(err, service.ErrUploadScanFailed):
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		default:
			h.logger.Error().Err(err).Msg("upload failed")
			return utils.SendError(c, fiber.StatusInternalServerError, "upload failed")
		}
	}

	return utils.SendSuccess(c, "upload successful", result)
}
