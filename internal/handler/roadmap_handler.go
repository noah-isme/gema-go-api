package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// RoadmapHandler exposes roadmap stage endpoints.
type RoadmapHandler struct {
	service service.RoadmapService
	logger  zerolog.Logger
}

// NewRoadmapHandler constructs a roadmap handler.
func NewRoadmapHandler(service service.RoadmapService, logger zerolog.Logger) *RoadmapHandler {
	return &RoadmapHandler{
		service: service,
		logger:  logger.With().Str("component", "roadmap_handler").Logger(),
	}
}

// Register wires roadmap routes.
func (h *RoadmapHandler) Register(router fiber.Router) {
	router.Get("/stages", h.listStages)
}

func (h *RoadmapHandler) listStages(c *fiber.Ctx) error {
	page, err := parseQueryInt(c, "page")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid page")
	}
	pageSize, err := parseQueryInt(c, "pageSize")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid page size")
	}
	if pageSize == 0 {
		if legacy, legacyErr := parseQueryInt(c, "page_size"); legacyErr == nil {
			pageSize = legacy
		}
	}

	req := dto.RoadmapStageListRequest{
		Page:     page,
		PageSize: pageSize,
		Sort:     c.Query("sort"),
		Search:   c.Query("search"),
		Tags:     splitAndTrim(c.Query("tags")),
	}

	result, err := h.service.ListStages(c.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list roadmap stages")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to fetch roadmap")
	}

	meta := fiber.Map{
		"pagination": result.Pagination,
		"filters":    result.Filters,
		"cache_hit":  result.CacheHit,
	}

	return utils.OK(c, result.Items, "roadmap stages retrieved", meta)
}
