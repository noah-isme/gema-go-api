package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/middleware"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
	"gorm.io/gorm"
)

// DiscussionHandler provides HTTP endpoints for discussion threads.
type DiscussionHandler struct {
	service   service.DiscussionService
	validator *validator.Validate
	logger    zerolog.Logger
}

// NewDiscussionHandler constructs a handler instance.
func NewDiscussionHandler(service service.DiscussionService, validator *validator.Validate, logger zerolog.Logger) *DiscussionHandler {
	return &DiscussionHandler{
		service:   service,
		validator: validator,
		logger:    logger.With().Str("component", "discussion_handler").Logger(),
	}
}

// Register binds the discussion routes.
func (h *DiscussionHandler) Register(router fiber.Router) {
	router.Get("/threads", h.listThreads)
	router.Post("/threads", h.createThread)
	router.Get("/threads/:id", h.getThread)
	router.Put("/threads/:id", h.updateThread)
	router.Delete("/threads/:id", h.deleteThread)

	router.Get("/replies", h.listReplies)
	router.Post("/replies", h.createReply)
}

func (h *DiscussionHandler) listThreads(c *fiber.Ctx) error {
	limit, err := parseQueryInt(c, "limit")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid limit")
	}
	offset, err := parseQueryInt(c, "offset")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid offset")
	}

	ctx := withRequestContext(c)

	threads, err := h.service.ListThreads(ctx, limit, offset)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SendSuccess(c, "threads", threads)
}

func (h *DiscussionHandler) getThread(c *fiber.Ctx) error {
	id, err := parseUintParamValue(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	includeReplies := strings.ToLower(strings.TrimSpace(c.Query("include_replies"))) == "true"

	ctx := withRequestContext(c)

	thread, err := h.service.GetThread(ctx, uint(id), includeReplies)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SendSuccess(c, "thread", thread)
}

func (h *DiscussionHandler) createThread(c *fiber.Ctx) error {
	userID := userIDStringFromContext(c)
	if userID == "" {
		return utils.SendError(c, fiber.StatusUnauthorized, "user not authenticated")
	}

	var payload dto.DiscussionThreadCreateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	ctx := withRequestContext(c)

	response, err := h.service.CreateThread(ctx, userID, userRoleFromContext(c), payload)
	if err != nil {
		status := fiber.StatusInternalServerError
		if isValidationError(err) {
			status = fiber.StatusBadRequest
		}
		return utils.SendError(c, status, err.Error())
	}

	return utils.SendSuccessWithStatus(c, fiber.StatusCreated, "thread created", response)
}

func (h *DiscussionHandler) updateThread(c *fiber.Ctx) error {
	userID := userIDStringFromContext(c)
	if userID == "" {
		return utils.SendError(c, fiber.StatusUnauthorized, "user not authenticated")
	}

	id, err := parseUintParamValue(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var payload dto.DiscussionThreadUpdateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	ctx := withRequestContext(c)

	response, err := h.service.UpdateThread(ctx, uint(id), userID, userRoleFromContext(c), payload)
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, service.ErrDiscussionForbidden) {
			status = fiber.StatusForbidden
		} else if isValidationError(err) {
			status = fiber.StatusBadRequest
		}
		return utils.SendError(c, status, err.Error())
	}

	return utils.SendSuccess(c, "thread updated", response)
}

func (h *DiscussionHandler) deleteThread(c *fiber.Ctx) error {
	userID := userIDStringFromContext(c)
	if userID == "" {
		return utils.SendError(c, fiber.StatusUnauthorized, "user not authenticated")
	}

	id, err := parseUintParamValue(c, "id")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	ctx := withRequestContext(c)

	if err := h.service.DeleteThread(ctx, uint(id), userID, userRoleFromContext(c)); err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, service.ErrDiscussionForbidden) {
			status = fiber.StatusForbidden
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = fiber.StatusNotFound
		}
		return utils.SendError(c, status, err.Error())
	}

	return utils.SendSuccess(c, "thread deleted", nil)
}

func (h *DiscussionHandler) listReplies(c *fiber.Ctx) error {
	threadIDParam := c.Query("thread_id")
	if threadIDParam == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "thread_id required")
	}

	threadID, err := strconv.ParseUint(threadIDParam, 10, 64)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid thread_id")
	}

	limit, err := parseQueryInt(c, "limit")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid limit")
	}
	offset, err := parseQueryInt(c, "offset")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid offset")
	}

	ctx := withRequestContext(c)

	replies, err := h.service.ListReplies(ctx, uint(threadID), limit, offset)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SendSuccess(c, "replies", replies)
}

func (h *DiscussionHandler) createReply(c *fiber.Ctx) error {
	userID := userIDStringFromContext(c)
	if userID == "" {
		return utils.SendError(c, fiber.StatusUnauthorized, "user not authenticated")
	}

	var payload dto.DiscussionReplyCreateRequest
	if err := c.BodyParser(&payload); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid payload")
	}

	ctx := withRequestContext(c)

	reply, err := h.service.CreateReply(ctx, userID, userRoleFromContext(c), payload)
	if err != nil {
		status := fiber.StatusInternalServerError
		if isValidationError(err) {
			status = fiber.StatusBadRequest
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			status = fiber.StatusNotFound
		}
		return utils.SendError(c, status, err.Error())
	}

	return utils.SendSuccessWithStatus(c, fiber.StatusCreated, "reply created", reply)
}

func parseUintParamValue(c *fiber.Ctx, key string) (uint64, error) {
	value := strings.TrimSpace(c.Params(key))
	if value == "" {
		return 0, fmt.Errorf("%s required", key)
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return parsed, nil
}

func withRequestContext(c *fiber.Ctx) context.Context {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}
	return middleware.ContextWithCorrelation(ctx, middleware.GetCorrelationID(c))
}
