package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AnnouncementHandler handles public announcement endpoints.
type AnnouncementHandler struct {
	service service.AnnouncementService
	logger  zerolog.Logger
}

// NewAnnouncementHandler constructs the handler.
func NewAnnouncementHandler(service service.AnnouncementService, logger zerolog.Logger) *AnnouncementHandler {
	return &AnnouncementHandler{
		service: service,
		logger:  logger.With().Str("component", "announcement_handler").Logger(),
	}
}

// Register wires routes for announcements.
func (h *AnnouncementHandler) Register(router fiber.Router) {
	router.Get("", h.list)
}

func (h *AnnouncementHandler) list(c *fiber.Ctx) error {
	page, err := parseQueryInt(c, "page")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid page")
	}
	if page <= 0 {
		page = 1
	}
	pageSize, err := parseQueryInt(c, "pageSize")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid page size")
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	result, err := h.service.ListActive(c.Context(), page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list announcements")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to list announcements")
	}

	if result.CacheHit {
		c.Set("X-Cache-Hit", "true")
	} else {
		c.Set("X-Cache-Hit", "false")
	}

	return utils.SendSuccess(c, "announcements retrieved", result)
}
