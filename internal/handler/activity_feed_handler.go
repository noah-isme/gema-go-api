package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// ActivityFeedHandler serves the public activity endpoints.
type ActivityFeedHandler struct {
	service service.ActivityFeedService
	logger  zerolog.Logger
}

// NewActivityFeedHandler constructs the handler instance.
func NewActivityFeedHandler(service service.ActivityFeedService, logger zerolog.Logger) *ActivityFeedHandler {
	return &ActivityFeedHandler{
		service: service,
		logger:  logger.With().Str("component", "activity_feed_handler").Logger(),
	}
}

// Register wires the activity feed routes.
func (h *ActivityFeedHandler) Register(router fiber.Router) {
	router.Get("/active", h.active)
}

func (h *ActivityFeedHandler) active(c *fiber.Ctx) error {
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
	var userID *uint
	if v := c.Query("userId"); v != "" {
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
			val := uint(parsed)
			userID = &val
		}
	}
	req := dto.ActivityFeedRequest{
		Page:     page,
		PageSize: pageSize,
		UserID:   userID,
		Type:     c.Query("type"),
		Action:   c.Query("action"),
	}

	result, err := h.service.ListActive(c.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to fetch active activities")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to fetch activities")
	}

	if result.CacheHit {
		c.Set("X-Cache-Hit", "true")
	} else {
		c.Set("X-Cache-Hit", "false")
	}

	return utils.SendSuccess(c, "active activities retrieved", result)
}
