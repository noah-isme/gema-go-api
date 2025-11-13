package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AdminGalleryHandler manages gallery admin endpoints.
type AdminGalleryHandler struct {
	service service.AdminGalleryService
	logger  zerolog.Logger
}

// NewAdminGalleryHandler constructs the handler.
func NewAdminGalleryHandler(service service.AdminGalleryService, logger zerolog.Logger) *AdminGalleryHandler {
	return &AdminGalleryHandler{
		service: service,
		logger:  logger.With().Str("component", "admin_gallery_handler").Logger(),
	}
}

// Register attaches routes.
func (h *AdminGalleryHandler) Register(router fiber.Router) {
	router.Get("", h.list)
	router.Post("", h.create)
	router.Patch("/:id", h.update)
	router.Delete("/:id", h.delete)
}

func (h *AdminGalleryHandler) list(c *fiber.Ctx) error {
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

	req := dto.AdminGalleryListRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   c.Query("search"),
		Tags:     splitAndTrim(c.Query("tags")),
	}

	result, err := h.service.List(c.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list gallery items (admin)")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to list gallery items")
	}

	meta := fiber.Map{
		"pagination": result.Pagination,
		"filters": fiber.Map{
			"search": req.Search,
			"tags":   req.Tags,
		},
	}

	return utils.OK(c, result.Items, "gallery items retrieved", meta)
}

func (h *AdminGalleryHandler) create(c *fiber.Ctx) error {
	var payload dto.AdminGalleryRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	actor := activityActorFromContext(c)
	item, err := h.service.Create(c.Context(), payload, actor)
	if err != nil {
		if isValidationError(err) {
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		}
		h.logger.Error().Err(err).Msg("failed to create gallery item")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to create gallery item")
	}

	return utils.SendSuccessWithStatus(c, fiber.StatusCreated, "gallery item created", item)
}

func (h *AdminGalleryHandler) update(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid identifier")
	}
	var payload dto.AdminGalleryRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}
	actor := activityActorFromContext(c)

	item, err := h.service.Update(c.Context(), id, payload, actor)
	if err != nil {
		switch {
		case isValidationError(err):
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrAdminGalleryNotFound):
			return utils.SendError(c, fiber.StatusNotFound, "gallery item not found")
		default:
			h.logger.Error().Err(err).Uint("gallery_id", id).Msg("failed to update gallery item")
			return utils.SendError(c, fiber.StatusInternalServerError, "failed to update gallery item")
		}
	}

	return utils.SendSuccess(c, "gallery item updated", item)
}

func (h *AdminGalleryHandler) delete(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid identifier")
	}
	actor := activityActorFromContext(c)

	if err := h.service.Delete(c.Context(), id, actor); err != nil {
		if errors.Is(err, service.ErrAdminGalleryNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "gallery item not found")
		}
		h.logger.Error().Err(err).Uint("gallery_id", id).Msg("failed to delete gallery item")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to delete gallery item")
	}

	return utils.SendSuccess(c, "gallery item deleted", fiber.Map{"id": id})
}
