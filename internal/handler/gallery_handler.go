package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// GalleryHandler exposes public gallery endpoints.
type GalleryHandler struct {
	service service.GalleryService
	logger  zerolog.Logger
}

// NewGalleryHandler constructs a gallery handler.
func NewGalleryHandler(service service.GalleryService, logger zerolog.Logger) *GalleryHandler {
	return &GalleryHandler{
		service: service,
		logger:  logger.With().Str("component", "gallery_handler").Logger(),
	}
}

// Register wires gallery routes.
func (h *GalleryHandler) Register(router fiber.Router) {
	router.Get("", h.list)
}

func (h *GalleryHandler) list(c *fiber.Ctx) error {
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
	tags := parseTags(c.Query("tags"))
	search := strings.TrimSpace(c.Query("search"))

	result, err := h.service.List(c.Context(), tags, search, page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list gallery items")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to list gallery items")
	}

	return utils.SendSuccess(c, "gallery items retrieved", result)
}

func parseTags(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		tags = append(tags, trimmed)
	}
	return tags
}
