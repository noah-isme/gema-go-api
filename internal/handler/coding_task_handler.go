package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// CodingTaskHandler exposes coding task HTTP endpoints.
type CodingTaskHandler struct {
	service service.CodingTaskService
	logger  zerolog.Logger
}

// NewCodingTaskHandler builds a new coding task handler.
func NewCodingTaskHandler(service service.CodingTaskService, logger zerolog.Logger) *CodingTaskHandler {
	return &CodingTaskHandler{
		service: service,
		logger:  logger.With().Str("component", "coding_task_handler").Logger(),
	}
}

// Register wires the handler routes into the router group.
func (h *CodingTaskHandler) Register(router fiber.Router) {
	router.Get("", h.list)
	router.Get("/:id", h.get)
}

func (h *CodingTaskHandler) list(c *fiber.Ctx) error {
	filter := dto.CodingTaskFilter{
		Language:   c.Query("language"),
		Difficulty: c.Query("difficulty"),
		Search:     c.Query("search"),
	}

	if tags := c.Query("tags"); tags != "" {
		filter.Tags = splitAndTrim(tags)
	}

	if page, err := parseQueryInt(c, "page"); err == nil {
		filter.Page = page
	}
	if pageSize, err := parseQueryInt(c, "page_size"); err == nil {
		filter.PageSize = pageSize
	}

	tasks, err := h.service.List(c.Context(), filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list coding tasks")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to retrieve tasks")
	}

	return utils.SendSuccess(c, "coding tasks retrieved", tasks)
}

func (h *CodingTaskHandler) get(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	task, err := h.service.Get(c.Context(), id)
	if err != nil {
		if err == service.ErrCodingTaskNotFound {
			return utils.SendError(c, fiber.StatusNotFound, "coding task not found")
		}
		h.logger.Error().Err(err).Uint("task_id", id).Msg("failed to get coding task")
		return utils.SendError(c, fiber.StatusInternalServerError, "failed to retrieve task")
	}

	return utils.SendSuccess(c, "coding task retrieved", task)
}
