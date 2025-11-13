package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AdminAnnouncementHandler manages admin announcement routes.
type AdminAnnouncementHandler struct {
	service service.AdminAnnouncementService
	logger  zerolog.Logger
}

// NewAdminAnnouncementHandler constructs the handler.
func NewAdminAnnouncementHandler(service service.AdminAnnouncementService, logger zerolog.Logger) *AdminAnnouncementHandler {
	return &AdminAnnouncementHandler{
		service: service,
		logger:  logger.With().Str("component", "admin_announcement_handler").Logger(),
	}
}

// Register attaches routes.
func (h *AdminAnnouncementHandler) Register(router fiber.Router) {
	router.Get("", h.list)
	router.Post("", h.create)
}

func (h *AdminAnnouncementHandler) list(c *fiber.Ctx) error {
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

	req := dto.AdminAnnouncementListRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   c.Query("search"),
	}

	result, err := h.service.List(c.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list announcements")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to list announcements")
	}

	meta := fiber.Map{"pagination": result.Pagination, "filters": fiber.Map{"search": req.Search}}
	return utils.OK(c, result.Items, "announcements retrieved", meta)
}

func (h *AdminAnnouncementHandler) create(c *fiber.Ctx) error {
	var payload dto.AdminAnnouncementRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	actor := activityActorFromContext(c)
	announcement, err := h.service.Create(c.Context(), payload, actor)
	if err != nil {
		if isValidationError(err) {
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		}
		h.logger.Error().Err(err).Msg("failed to create announcement")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to create announcement")
	}

	return utils.SendSuccessWithStatus(c, fiber.StatusCreated, "announcement created", announcement)
}
