package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/middleware"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/observability"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

const (
	chatRedisTTL       = 30 * time.Minute
	chatSendBufferSize = 32
)

// ErrChatNotAuthorised indicates the sender attempted to post into a room they do not control.
var ErrChatNotAuthorised = errors.New("sender not authorised for room")

// ChatConnectionOptions wraps metadata extracted during the HTTP upgrade.
type ChatConnectionOptions struct {
	UserID        string
	Role          string
	RoomID        string
	CorrelationID string
	Context       context.Context
}

// ChatService manages websocket chat connections and message delivery.
type ChatService interface {
	ServeConnection(conn *websocket.Conn, opts ChatConnectionOptions)
	History(ctx context.Context, query dto.ChatHistoryQuery) ([]dto.ChatMessageResponse, error)
	Start(ctx context.Context)
}

type chatService struct {
	repo        repository.ChatRepository
	redis       *redis.Client
	redisStream string
	redisCache  string
	nats        *nats.Conn
	natsSubject string
	validator   *validator.Validate
	logger      zerolog.Logger
	tracer      trace.Tracer
	sanitizer   *bluemonday.Policy
	hub         *chatHub
	nodeID      string
}

// chatHub keeps track of active websocket clients and handles broadcasting.
type chatHub struct {
	mu    sync.RWMutex
	rooms map[string]map[*chatClient]struct{}
	log   zerolog.Logger
}

type chatClient struct {
	conn          *websocket.Conn
	send          chan dto.ChatMessageResponse
	options       ChatConnectionOptions
	service       *chatService
	closed        chan struct{}
	once          sync.Once
	lastHeartbeat time.Time
	baseCtx       context.Context
}

type chatEvent struct {
	Source   string                  `json:"source"`
	Message  dto.ChatMessageResponse `json:"message"`
	SentAt   time.Time               `json:"sent_at"`
	Metadata map[string]string       `json:"metadata,omitempty"`
}

// NewChatService creates a websocket chat service instance.
func NewChatService(repo repository.ChatRepository, redisClient *redis.Client, channelBase string, natsConn *nats.Conn, validate *validator.Validate, logger zerolog.Logger) ChatService {
	sanitizer := bluemonday.UGCPolicy()
	sanitizer.AllowElements("br")

	hub := &chatHub{
		rooms: make(map[string]map[*chatClient]struct{}),
		log:   logger.With().Str("component", "chat_hub").Logger(),
	}

	tracer := otel.Tracer("github.com/noah-isme/gema-go-api/internal/service/chat")

	streamChannel := ""
	cachePrefix := ""
	natsSubject := ""
	if channelBase != "" {
		streamChannel = channelBase + ":chat"
		cachePrefix = channelBase + ":chat:last"
		natsSubject = strings.ReplaceAll(channelBase, ":", ".") + ".chat"
	}

	return &chatService{
		repo:        repo,
		redis:       redisClient,
		redisStream: streamChannel,
		redisCache:  cachePrefix,
		nats:        natsConn,
		natsSubject: natsSubject,
		validator:   validate,
		logger:      logger.With().Str("component", "chat_service").Logger(),
		tracer:      tracer,
		sanitizer:   sanitizer,
		hub:         hub,
		nodeID:      uuid.NewString(),
	}
}

func (s *chatService) Start(ctx context.Context) {
	if s.redis != nil && s.redisStream != "" {
		go s.consumeRedis(ctx)
	}
	if s.nats != nil && s.natsSubject != "" {
		go s.consumeNATS(ctx)
	}
}

func (s *chatService) ServeConnection(conn *websocket.Conn, opts ChatConnectionOptions) {
	baseCtx := opts.Context
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	client := &chatClient{
		conn:    conn,
		send:    make(chan dto.ChatMessageResponse, chatSendBufferSize),
		options: opts,
		service: s,
		closed:  make(chan struct{}),
		baseCtx: baseCtx,
	}

	s.hub.register(client)
	observability.ChatConnectionsTotal().Inc()

	if last := s.fetchLastMessage(baseCtx, opts.RoomID); last != nil {
		select {
		case client.send <- *last:
		default:
			s.logger.Debug().Str("room_id", opts.RoomID).Msg("dropping cached chat message due to slow consumer")
		}
	}

	go client.writer()
	client.reader()
}

func (s *chatService) History(ctx context.Context, query dto.ChatHistoryQuery) ([]dto.ChatMessageResponse, error) {
	if err := s.validator.Struct(query); err != nil {
		return nil, err
	}

	before := time.Time{}
	if query.Before != nil {
		before = *query.Before
	}

	messages, err := s.repo.ListByRoom(ctx, query.RoomID, before, query.Limit)
	if err != nil {
		return nil, err
	}

	return dto.NewChatMessageResponseSlice(messages), nil
}

func (s *chatService) processSend(ctx context.Context, client *chatClient, correlation string, payload dto.ChatSendRequest) (dto.ChatMessageResponse, error) {
	if payload.RoomID == "" {
		payload.RoomID = client.options.RoomID
	}

	payload.RoomID = strings.TrimSpace(payload.RoomID)
	payload.ReceiverID = strings.TrimSpace(payload.ReceiverID)

	if err := s.validator.Struct(payload); err != nil {
		return dto.ChatMessageResponse{}, err
	}

	if err := s.authorise(client, payload); err != nil {
		return dto.ChatMessageResponse{}, err
	}

	clean := strings.TrimSpace(s.sanitizer.Sanitize(payload.Content))
	if clean == "" {
		return dto.ChatMessageResponse{}, fmt.Errorf("message content empty after sanitization")
	}

	messageType := payload.Type
	if messageType == "" {
		messageType = "text"
	}

	attrs := []attribute.KeyValue{
		attribute.String("chat.room_id", payload.RoomID),
		attribute.String("chat.sender_id", client.options.UserID),
		attribute.String("chat.type", messageType),
	}
	if correlation != "" {
		attrs = append(attrs, attribute.String("correlation_id", correlation))
	}

	spanCtx, span := s.tracer.Start(ctx, "chat.broadcast", trace.WithAttributes(attrs...))
	defer span.End()

	model := models.ChatMessage{
		SenderID:   client.options.UserID,
		ReceiverID: payload.ReceiverID,
		RoomID:     payload.RoomID,
		Content:    clean,
		Type:       messageType,
	}

	if err := s.repo.Save(spanCtx, &model); err != nil {
		span.RecordError(err)
		return dto.ChatMessageResponse{}, err
	}

	response := dto.NewChatMessageResponse(model)
	s.cacheLastMessage(spanCtx, response)
	s.broadcast(response)
	if err := s.publish(spanCtx, response); err != nil {
		s.logger.Warn().Err(err).Msg("failed to publish chat event")
	}

	observability.ChatMessagesSent().WithLabelValues(messageType).Inc()

	return response, nil
}

func (s *chatService) authorise(client *chatClient, payload dto.ChatSendRequest) error {
	role := strings.ToLower(client.options.Role)
	switch role {
	case "admin", "teacher":
		return nil
	case "student":
		if strings.Contains(payload.RoomID, client.options.UserID) {
			return nil
		}
		if payload.ReceiverID != "" && payload.ReceiverID == client.options.UserID {
			return nil
		}
		return ErrChatNotAuthorised
	default:
		if payload.ReceiverID == client.options.UserID && payload.RoomID == client.options.RoomID {
			return nil
		}
		return ErrChatNotAuthorised
	}
}

func (s *chatService) cacheLastMessage(ctx context.Context, message dto.ChatMessageResponse) {
	if s.redis == nil || s.redisCache == "" {
		return
	}

	payload, err := json.Marshal(message)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to marshal chat message for cache")
		return
	}

	key := fmt.Sprintf("%s:%s", s.redisCache, message.RoomID)
	if err := s.redis.Set(ctx, key, payload, chatRedisTTL).Err(); err != nil {
		s.logger.Warn().Err(err).Msg("failed to cache chat message")
	}
}

func (s *chatService) fetchLastMessage(ctx context.Context, roomID string) *dto.ChatMessageResponse {
	if s.redis == nil || s.redisCache == "" {
		return nil
	}

	key := fmt.Sprintf("%s:%s", s.redisCache, roomID)
	result, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil
	}

	var message dto.ChatMessageResponse
	if err := json.Unmarshal([]byte(result), &message); err != nil {
		s.logger.Warn().Err(err).Msg("failed to unmarshal cached chat message")
		return nil
	}

	return &message
}

func (s *chatService) broadcast(message dto.ChatMessageResponse) {
	s.hub.broadcast(message.RoomID, message)
}

func (s *chatService) publish(ctx context.Context, message dto.ChatMessageResponse) error {
	event := chatEvent{
		Source:  s.nodeID,
		Message: message,
		SentAt:  time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if s.redis != nil && s.redisStream != "" {
		if err := s.redis.Publish(ctx, s.redisStream, payload).Err(); err != nil {
			return err
		}
	}

	if s.nats != nil && s.natsSubject != "" {
		if err := s.nats.Publish(s.natsSubject, payload); err != nil {
			return err
		}
	}

	return nil
}

func (s *chatService) consumeRedis(ctx context.Context) {
	pubsub := s.redis.Subscribe(ctx, s.redisStream)
	defer func() {
		_ = pubsub.Close()
	}()
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			s.logger.Error().Err(err).Msg("chat redis subscription closed")
			return
		}
		s.handleEvent([]byte(msg.Payload))
	}
}

func (s *chatService) consumeNATS(ctx context.Context) {
	sub, err := s.nats.QueueSubscribe(s.natsSubject, "gema-chat", func(msg *nats.Msg) {
		s.handleEvent(msg.Data)
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to subscribe to nats chat subject")
		return
	}
	go func() {
		<-ctx.Done()
		if err := sub.Drain(); err != nil {
			s.logger.Warn().Err(err).Msg("failed to drain chat nats subscription")
		}
	}()
}

func (s *chatService) handleEvent(data []byte) {
	var event chatEvent
	if err := json.Unmarshal(data, &event); err != nil {
		s.logger.Warn().Err(err).Msg("invalid chat event")
		return
	}

	if event.Source == s.nodeID {
		return
	}

	messageType := event.Message.Type
	if messageType == "" {
		messageType = "text"
	}

	observability.ChatMessagesSent().WithLabelValues(messageType).Inc()
	s.broadcast(event.Message)
}

func (h *chatHub) register(client *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room := client.options.RoomID
	if room == "" {
		room = "default"
	}

	if _, exists := h.rooms[room]; !exists {
		h.rooms[room] = make(map[*chatClient]struct{})
	}
	client.options.RoomID = room
	h.rooms[room][client] = struct{}{}
	h.log.Debug().Str("room_id", room).Str("user_id", client.options.UserID).Msg("chat client connected")
}

func (h *chatHub) unregister(client *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room := client.options.RoomID
	if clients, ok := h.rooms[room]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.rooms, room)
		}
	}
	h.log.Debug().Str("room_id", room).Str("user_id", client.options.UserID).Msg("chat client disconnected")
}

func (h *chatHub) broadcast(roomID string, message dto.ChatMessageResponse) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := h.rooms[roomID]
	for client := range clients {
		select {
		case client.send <- message:
		default:
			h.log.Warn().Str("room_id", roomID).Str("user_id", client.options.UserID).Msg("dropping chat message for slow client")
		}
	}
}

func (c *chatClient) reader() {
	defer c.close()

	connCtx := c.baseCtx
	if connCtx == nil {
		connCtx = context.Background()
	}
	correlation := c.options.CorrelationID
	if correlation == "" {
		correlation = middleware.CorrelationIDFromContext(connCtx)
	}

	for {
		var payload dto.ChatSendRequest
		if err := c.conn.ReadJSON(&payload); err != nil {
			c.service.logger.Debug().Err(err).Msg("chat read loop ended")
			return
		}

		response, err := c.service.processSend(connCtx, c, correlation, payload)
		if err != nil {
			c.service.logger.Warn().Err(err).Msg("failed to process chat message")
			continue
		}

		select {
		case <-c.closed:
			return
		default:
		}

		select {
		case c.send <- response:
		default:
			c.service.logger.Warn().Msg("sender queue full, dropping ack message")
		}
	}
}

func (c *chatClient) writer() {
	defer c.close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.WriteJSON(message); err != nil {
				c.service.logger.Debug().Err(err).Msg("chat write loop terminated")
				return
			}
		case <-time.After(30 * time.Second):
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte("keepalive")); err != nil {
				c.service.logger.Debug().Err(err).Msg("chat ping failed")
				return
			}
		case <-c.closed:
			return
		}
	}
}

func (c *chatClient) close() {
	c.once.Do(func() {
		close(c.closed)
		c.service.hub.unregister(c)
		_ = c.conn.Close()
	})
}
