package handler

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// ContactHandler handles contact submissions.
type ContactHandler struct {
	service service.ContactService
	logger  zerolog.Logger
}

// NewContactHandler constructs a contact handler.
func NewContactHandler(service service.ContactService, logger zerolog.Logger) *ContactHandler {
	return &ContactHandler{
		service: service,
		logger:  logger.With().Str("component", "contact_handler").Logger(),
	}
}

// Register wires contact routes.
func (h *ContactHandler) Register(router fiber.Router) {
	router.Post("", h.submit)
}

func (h *ContactHandler) submit(c *fiber.Ctx) error {
	var payload dto.ContactRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	if payload.Honeypot != "" {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	payload.IPAddress = c.IP()
	if userID, ok := c.Locals("user_id").(uint); ok && userID > 0 {
		payload.UserID = &userID
	}

	response, err := h.service.Submit(c.Context(), payload)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrContactSpam):
			return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
		case errors.Is(err, service.ErrContactDuplicate):
			return utils.SendError(c, fiber.StatusTooManyRequests, "duplicate submission")
		default:
			h.logger.Error().Err(err).Msg("failed to process contact submission")
			return utils.SendError(c, fiber.StatusInternalServerError, "failed to submit contact form")
		}
	}

	return utils.SendSuccess(c, "contact submission accepted", response)
}
