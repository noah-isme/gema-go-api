package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AdminAnalyticsHandler exposes analytics endpoints for administrators.
type AdminAnalyticsHandler struct {
	service service.AdminAnalyticsService
	logger  zerolog.Logger
}

// NewAdminAnalyticsHandler constructs the handler.
func NewAdminAnalyticsHandler(service service.AdminAnalyticsService, logger zerolog.Logger) *AdminAnalyticsHandler {
	return &AdminAnalyticsHandler{
		service: service,
		logger:  logger.With().Str("component", "admin_analytics_handler").Logger(),
	}
}

// Register attaches analytics routes to the router group.
func (h *AdminAnalyticsHandler) Register(router fiber.Router) {
	router.Get("", h.get)
}

func (h *AdminAnalyticsHandler) get(c *fiber.Ctx) error {
	summary, err := h.service.GetSummary(c.Context())
	if err != nil {
		requestLogger(h.logger, c).Error().Err(err).Msg("failed to fetch analytics summary")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to load analytics")
	}

	return utils.SendSuccess(c, "analytics summary", summary)
}
