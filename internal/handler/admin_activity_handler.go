package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AdminActivityHandler exposes activity log endpoints.
type AdminActivityHandler struct {
	service service.ActivityService
	logger  zerolog.Logger
}

// NewAdminActivityHandler constructs the handler.
func NewAdminActivityHandler(service service.ActivityService, logger zerolog.Logger) *AdminActivityHandler {
	return &AdminActivityHandler{
		service: service,
		logger:  logger.With().Str("component", "admin_activity_handler").Logger(),
	}
}

// Register attaches activity log routes to the router group.
func (h *AdminActivityHandler) Register(router fiber.Router) {
	router.Get("", h.list)
	router.Post("", h.create)
}

func (h *AdminActivityHandler) list(c *fiber.Ctx) error {
	page, err := parseQueryInt(c, "page")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid page")
	}
	if page <= 0 {
		page = 1
	}

	pageSize, err := parseQueryInt(c, "page_size")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid page size")
	}
	if pageSize <= 0 {
		pageSize = 25
	} else if pageSize > 200 {
		pageSize = 200
	}

	actorIDInt, err := parseQueryInt(c, "actor_id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid actor id")
	}

	req := dto.AdminActivityListRequest{
		Page:       page,
		PageSize:   pageSize,
		Action:     c.Query("action"),
		EntityType: c.Query("entity_type"),
	}
	if actorIDInt > 0 {
		req.ActorID = uint(actorIDInt)
	}

	response, err := h.service.List(c.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list activity logs")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to list activity logs")
	}

	return utils.SendSuccess(c, "activity logs", response)
}

func (h *AdminActivityHandler) create(c *fiber.Ctx) error {
	var payload dto.AdminActivityCreateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	actor := activityActorFromContext(c)
	entry, err := h.service.Create(c.Context(), actor, payload)
	if err != nil {
		if isValidationError(err) {
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		}
		h.logger.Error().Err(err).Msg("failed to create activity log")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to create activity log")
	}

	return utils.SendSuccessWithStatus(c, fiber.StatusCreated, "activity log created", entry)
}
