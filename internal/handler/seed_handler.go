package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// SeedHandler exposes tooling endpoints for seeding data.
type SeedHandler struct {
	service service.SeedService
	logger  zerolog.Logger
}

// NewSeedHandler constructs a seed handler.
func NewSeedHandler(service service.SeedService, logger zerolog.Logger) *SeedHandler {
	return &SeedHandler{
		service: service,
		logger:  logger.With().Str("component", "seed_handler").Logger(),
	}
}

// Register wires seed routes.
func (h *SeedHandler) Register(router fiber.Router) {
	router.Post("/announcements", h.announcements)
	router.Post("/gallery", h.gallery)
}

type seedAnnouncementsRequest struct {
	Items []models.Announcement `json:"items"`
}

type seedGalleryRequest struct {
	Items []models.GalleryItem `json:"items"`
}

func (h *SeedHandler) announcements(c *fiber.Ctx) error {
	token := c.Get("X-Seed-Token")
	var payload seedAnnouncementsRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	affected, err := h.service.SeedAnnouncements(c.Context(), token, payload.Items)
	if err != nil {
		return h.seedError(c, err)
	}

	return utils.SendSuccess(c, "announcements seeded", fiber.Map{"affected": affected})
}

func (h *SeedHandler) gallery(c *fiber.Ctx) error {
	token := c.Get("X-Seed-Token")
	var payload seedGalleryRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	affected, err := h.service.SeedGallery(c.Context(), token, payload.Items)
	if err != nil {
		return h.seedError(c, err)
	}

	return utils.SendSuccess(c, "gallery seeded", fiber.Map{"affected": affected})
}

func (h *SeedHandler) seedError(c *fiber.Ctx, err error) error {
	switch err {
	case service.ErrSeedDisabled:
		return utils.SendError(c, fiber.StatusForbidden, "seeding disabled")
	case service.ErrSeedUnauthorized:
		return utils.SendError(c, fiber.StatusForbidden, "invalid token")
	default:
		h.logger.Error().Err(err).Msg("seed operation failed")
		return utils.SendError(c, fiber.StatusInternalServerError, "seed operation failed")
	}
}
