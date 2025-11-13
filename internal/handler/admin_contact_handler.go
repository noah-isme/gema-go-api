package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// AdminContactHandler exposes admin contact endpoints.
type AdminContactHandler struct {
	service service.AdminContactService
	logger  zerolog.Logger
}

// NewAdminContactHandler constructs the handler.
func NewAdminContactHandler(service service.AdminContactService, logger zerolog.Logger) *AdminContactHandler {
	return &AdminContactHandler{
		service: service,
		logger:  logger.With().Str("component", "admin_contact_handler").Logger(),
	}
}

// Register attaches routes.
func (h *AdminContactHandler) Register(router fiber.Router) {
	router.Get("", h.list)
	router.Get("/:id", h.get)
}

func (h *AdminContactHandler) list(c *fiber.Ctx) error {
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

	req := dto.AdminContactListRequest{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Search:   c.Query("search"),
		Sort:     c.Query("sort"),
	}

	result, err := h.service.List(c.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list contact submissions")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to list contacts")
	}

	meta := fiber.Map{
		"pagination": result.Pagination,
		"filters": fiber.Map{
			"status": req.Status,
			"search": req.Search,
			"sort":   req.Sort,
		},
	}

	return utils.OK(c, result.Items, "contact submissions retrieved", meta)
}

func (h *AdminContactHandler) get(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	submission, err := h.service.Get(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAdminContactNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "contact submission not found")
		}
		h.logger.Error().Err(err).Uint("contact_id", id).Msg("failed to fetch contact submission")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to fetch contact submission")
	}

	return utils.OK(c, submission, "contact submission retrieved", nil)
}
