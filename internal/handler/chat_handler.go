package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/middleware"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/internal/utils"
)

// ChatHandler wires chat endpoints including the websocket upgrade.
type ChatHandler struct {
	service   service.ChatService
	validator *validator.Validate
	logger    zerolog.Logger
}

// NewChatHandler creates a chat handler instance.
func NewChatHandler(service service.ChatService, validator *validator.Validate, logger zerolog.Logger) *ChatHandler {
	return &ChatHandler{
		service:   service,
		validator: validator,
		logger:    logger.With().Str("component", "chat_handler").Logger(),
	}
}

// Register binds chat routes under the provided router group.
func (h *ChatHandler) Register(router fiber.Router) {
	router.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			ctx := c.UserContext()
			if ctx == nil {
				ctx = context.Background()
			}
			ctx = middleware.ContextWithCorrelation(ctx, middleware.GetCorrelationID(c))
			c.Locals("request_ctx", ctx)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	router.Get("/ws", websocket.New(h.handleConnection))
	router.Get("/history", h.history)
}

func (h *ChatHandler) handleConnection(conn *websocket.Conn) {
	userID := websocketUserID(conn)
	if userID == "" {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(fiber.StatusUnauthorized, "user id missing"))
		_ = conn.Close()
		return
	}

	roomID := strings.TrimSpace(conn.Query("room_id"))
	if roomID == "" {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(fiber.StatusBadRequest, "room_id required"))
		_ = conn.Close()
		return
	}

	role := fmt.Sprint(conn.Locals("user_role"))
	correlation := fmt.Sprint(conn.Locals("correlation_id"))
	baseCtx, _ := conn.Locals("request_ctx").(context.Context)

	opts := service.ChatConnectionOptions{
		UserID:        userID,
		Role:          role,
		RoomID:        roomID,
		CorrelationID: correlation,
		Context:       baseCtx,
	}

	h.logger.Info().Str("user_id", userID).Str("room_id", roomID).Msg("chat websocket connected")
	h.service.ServeConnection(conn, opts)
	h.logger.Info().Str("user_id", userID).Str("room_id", roomID).Msg("chat websocket disconnected")
}

func (h *ChatHandler) history(c *fiber.Ctx) error {
	roomID := c.Query("room_id")
	if roomID == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "room_id required")
	}

	var beforePtr *time.Time
	if before := c.Query("before"); before != "" {
		parsed, err := time.Parse(time.RFC3339, before)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, "invalid before timestamp")
		}
		beforePtr = &parsed
	}

	limit := 0
	if limitRaw := c.Query("limit"); limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, "invalid limit")
		}
		limit = parsed
	}

	query := dto.ChatHistoryQuery{
		RoomID: roomID,
		Before: beforePtr,
		Limit:  limit,
	}

	if err := h.validator.Struct(query); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = middleware.ContextWithCorrelation(ctx, middleware.GetCorrelationID(c))

	messages, err := h.service.History(ctx, query)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SendSuccess(c, "chat history", messages)
}

func websocketUserID(conn *websocket.Conn) string {
	if value := conn.Locals("user_id"); value != nil {
		switch v := value.(type) {
		case float64:
			return fmt.Sprintf("%d", uint(v))
		case uint:
			return fmt.Sprintf("%d", v)
		case int:
			return fmt.Sprintf("%d", v)
		case string:
			return strings.TrimSpace(v)
		}
	}
	return ""
}
