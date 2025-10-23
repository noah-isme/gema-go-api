package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/middleware"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// NotificationHandler manages SSE notification streams and CRUD operations.
type NotificationHandler struct {
	service service.NotificationService
	logger  zerolog.Logger
	timeout time.Duration
}

// NewNotificationHandler constructs a handler instance.
func NewNotificationHandler(service service.NotificationService, logger zerolog.Logger, timeout time.Duration) *NotificationHandler {
	return &NotificationHandler{
		service: service,
		logger:  logger.With().Str("component", "notification_handler").Logger(),
		timeout: timeout,
	}
}

// Register binds the notification routes.
func (h *NotificationHandler) Register(router fiber.Router) {
	router.Get("/", h.list)
	router.Get("/stream", h.stream)
	router.Patch("/:id/read", h.markRead)
}

func (h *NotificationHandler) list(c *fiber.Ctx) error {
	userID := userIDStringFromContext(c)
	if userID == "" {
		return utils.SendError(c, fiber.StatusUnauthorized, "user not authenticated")
	}

	limit, err := parseQueryInt(c, "limit")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid limit")
	}
	offset, err := parseQueryInt(c, "offset")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid offset")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = middleware.ContextWithCorrelation(ctx, middleware.GetCorrelationID(c))

	notifications, err := h.service.List(ctx, userID, limit, offset)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SendSuccess(c, "notifications", notifications)
}

func (h *NotificationHandler) stream(c *fiber.Ctx) error {
	userID := userIDStringFromContext(c)
	if userID == "" {
		return utils.SendError(c, fiber.StatusUnauthorized, "user not authenticated")
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = middleware.ContextWithCorrelation(ctx, middleware.GetCorrelationID(c))
	ctx, cancel := context.WithCancel(ctx)

	stream, cleanup := h.service.Subscribe(userID)

	keepAliveInterval := h.timeout
	if keepAliveInterval <= 0 {
		keepAliveInterval = 30 * time.Second
	}

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer func() {
			cleanup()
			cancel()
		}()

		ticker := time.NewTicker(keepAliveInterval / 2)
		defer ticker.Stop()

		for {
			select {
			case notification, ok := <-stream:
				if !ok {
					return
				}
				if err := writeNotificationEvent(w, notification); err != nil {
					h.logger.Debug().Err(err).Msg("failed to write notification event")
					return
				}
			case <-ticker.C:
				if err := writeKeepAlive(w); err != nil {
					h.logger.Debug().Err(err).Msg("failed to write notification keepalive")
					return
				}
			case <-ctx.Done():
				return
			}
		}
	})

	return nil
}

func (h *NotificationHandler) markRead(c *fiber.Ctx) error {
	userID := userIDStringFromContext(c)
	if userID == "" {
		return utils.SendError(c, fiber.StatusUnauthorized, "user not authenticated")
	}

	idParam := c.Params("id")
	if idParam == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "notification id required")
	}

	parsed, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "invalid notification id")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = middleware.ContextWithCorrelation(ctx, middleware.GetCorrelationID(c))

	notification, err := h.service.MarkRead(ctx, uint(parsed), userID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SendSuccess(c, "notification updated", notification)
}

func writeNotificationEvent(w *bufio.Writer, notification interface{}) error {
	payload, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "event: notification\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	return w.Flush()
}

func writeKeepAlive(w *bufio.Writer) error {
	if _, err := fmt.Fprintf(w, ": keep-alive %s\n\n", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return w.Flush()
}
