package handler

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// TutorialContentHandler exposes tutorial articles & projects.
type TutorialContentHandler struct {
	service service.TutorialContentService
	logger  zerolog.Logger
}

// NewTutorialContentHandler constructs the handler.
func NewTutorialContentHandler(service service.TutorialContentService, logger zerolog.Logger) *TutorialContentHandler {
	return &TutorialContentHandler{
		service: service,
		logger:  logger.With().Str("component", "tutorial_content_handler").Logger(),
	}
}

// RegisterPublic wires readonly tutorial routes.
func (h *TutorialContentHandler) RegisterPublic(router fiber.Router) {
	router.Get("/articles", h.listArticles)
	router.Get("/articles/:id", h.getArticle)
	router.Get("/projects", h.listProjects)
	router.Get("/projects/:id", h.getProject)
}

// RegisterAdmin wires admin-only tutorial routes.
func (h *TutorialContentHandler) RegisterAdmin(router fiber.Router) {
	router.Post("/articles", h.createArticle)
	router.Post("/projects", h.createProject)
}

func (h *TutorialContentHandler) listArticles(c *fiber.Ctx) error {
	req, err := h.parseListRequest(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.ListArticles(c.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list tutorial articles")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to list articles")
	}

	meta := fiber.Map{
		"pagination": result.Pagination,
		"filters":    result.Filters,
	}

	return utils.OK(c, result.Items, "tutorial articles retrieved", meta)
}

func (h *TutorialContentHandler) listProjects(c *fiber.Ctx) error {
	req, err := h.parseListRequest(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.service.ListProjects(c.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list tutorial projects")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to list projects")
	}

	meta := fiber.Map{
		"pagination": result.Pagination,
		"filters":    result.Filters,
	}

	return utils.OK(c, result.Items, "tutorial projects retrieved", meta)
}

func (h *TutorialContentHandler) getArticle(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	article, err := h.service.GetArticle(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrTutorialArticleNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "article not found")
		}
		h.logger.Error().Err(err).Msg("failed to get tutorial article")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to fetch article")
	}

	return utils.OK(c, article, "tutorial article retrieved", nil)
}

func (h *TutorialContentHandler) getProject(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	project, err := h.service.GetProject(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrTutorialProjectNotFound) {
			return utils.SendError(c, fiber.StatusNotFound, "project not found")
		}
		h.logger.Error().Err(err).Msg("failed to get tutorial project")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to fetch project")
	}

	return utils.OK(c, project, "tutorial project retrieved", nil)
}

func (h *TutorialContentHandler) createArticle(c *fiber.Ctx) error {
	var payload dto.TutorialArticleCreateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	article, err := h.service.CreateArticle(c.Context(), payload)
	if err != nil {
		if isValidationError(err) {
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		}
		h.logger.Error().Err(err).Msg("failed to create tutorial article")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to create article")
	}

	return utils.SendSuccessWithStatus(c, fiber.StatusCreated, "tutorial article created", article)
}

func (h *TutorialContentHandler) createProject(c *fiber.Ctx) error {
	var payload dto.TutorialProjectCreateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	project, err := h.service.CreateProject(c.Context(), payload)
	if err != nil {
		if isValidationError(err) {
			return utils.SendError(c, fiber.StatusBadRequest, err.Error())
		}
		h.logger.Error().Err(err).Msg("failed to create tutorial project")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to create project")
	}

	return utils.SendSuccessWithStatus(c, fiber.StatusCreated, "tutorial project created", project)
}

func (h *TutorialContentHandler) parseListRequest(c *fiber.Ctx) (dto.TutorialContentListRequest, error) {
	page, err := parseQueryInt(c, "page")
	if err != nil {
		return dto.TutorialContentListRequest{}, err
	}

	pageSize, err := parseQueryInt(c, "pageSize")
	if err != nil {
		return dto.TutorialContentListRequest{}, err
	}
	if pageSize == 0 {
		if legacy, legacyErr := parseQueryInt(c, "page_size"); legacyErr == nil {
			pageSize = legacy
		}
	}

	tags := splitAndTrim(c.Query("tags"))

	return dto.TutorialContentListRequest{
		Page:     page,
		PageSize: pageSize,
		Sort:     c.Query("sort"),
		Search:   c.Query("search"),
		Tags:     tags,
	}, nil
}
