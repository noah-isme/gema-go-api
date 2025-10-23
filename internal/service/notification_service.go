package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/observability"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

const notificationBufferSize = 16

// NotificationService publishes and streams notifications to end users via SSE.
type NotificationService interface {
	Publish(ctx context.Context, payload dto.NotificationCreateRequest) (dto.NotificationResponse, error)
	List(ctx context.Context, userID string, limit, offset int) ([]dto.NotificationResponse, error)
	MarkRead(ctx context.Context, id uint, userID string) (dto.NotificationResponse, error)
	Subscribe(userID string) (<-chan dto.NotificationResponse, func())
	Start(ctx context.Context)
}

type notificationService struct {
	repo        repository.NotificationRepository
	redis       *redis.Client
	redisStream string
	nats        *nats.Conn
	natsSubject string
	validator   *validator.Validate
	logger      zerolog.Logger
	tracer      trace.Tracer
	sanitizer   *bluemonday.Policy
	broker      *notificationBroker
	nodeID      string
}

type notificationEvent struct {
	Source       string                   `json:"source"`
	Notification dto.NotificationResponse `json:"notification"`
	SentAt       time.Time                `json:"sent_at"`
}

type notificationBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan dto.NotificationResponse]struct{}
}

// NewNotificationService constructs a notification service.
func NewNotificationService(repo repository.NotificationRepository, redisClient *redis.Client, channelBase string, natsConn *nats.Conn, validate *validator.Validate, logger zerolog.Logger) NotificationService {
	stream := ""
	subject := ""
	if channelBase != "" {
		stream = channelBase + ":notifications"
		subject = strings.ReplaceAll(channelBase, ":", ".") + ".notifications"
	}

	return &notificationService{
		repo:        repo,
		redis:       redisClient,
		redisStream: stream,
		nats:        natsConn,
		natsSubject: subject,
		validator:   validate,
		logger:      logger.With().Str("component", "notification_service").Logger(),
		tracer:      otel.Tracer("github.com/noah-isme/gema-go-api/internal/service/notification"),
		sanitizer:   bluemonday.StrictPolicy(),
		broker: &notificationBroker{
			subscribers: make(map[string]map[chan dto.NotificationResponse]struct{}),
		},
		nodeID: uuid.NewString(),
	}
}

func (s *notificationService) Start(ctx context.Context) {
	if s.redis != nil && s.redisStream != "" {
		go s.consumeRedis(ctx)
	}
	if s.nats != nil && s.natsSubject != "" {
		go s.consumeNATS(ctx)
	}
}

func (s *notificationService) Publish(ctx context.Context, payload dto.NotificationCreateRequest) (dto.NotificationResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.NotificationResponse{}, err
	}

	cleanMessage := strings.TrimSpace(s.sanitizer.Sanitize(payload.Message))
	if cleanMessage == "" {
		return dto.NotificationResponse{}, errors.New("notification message empty after sanitization")
	}

	attrs := []attribute.KeyValue{
		attribute.String("notification.user_id", payload.UserID),
		attribute.String("notification.type", payload.Type),
	}

	spanCtx, span := s.tracer.Start(ctx, "notifications.publish", trace.WithAttributes(attrs...))
	defer span.End()

	model := models.Notification{
		UserID:  payload.UserID,
		Type:    payload.Type,
		Message: cleanMessage,
	}

	if err := s.repo.Create(spanCtx, &model); err != nil {
		span.RecordError(err)
		return dto.NotificationResponse{}, err
	}

	response := dto.NewNotificationResponse(model)
	s.broadcast(response)
	if err := s.publish(spanCtx, response); err != nil {
		s.logger.Warn().Err(err).Msg("failed to publish notification to broker")
	}

	observability.NotificationsPublishedTotal().WithLabelValues(response.Type).Inc()

	return response, nil
}

func (s *notificationService) List(ctx context.Context, userID string, limit, offset int) ([]dto.NotificationResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	notifications, err := s.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	return dto.NewNotificationResponseSlice(notifications), nil
}

func (s *notificationService) MarkRead(ctx context.Context, id uint, userID string) (dto.NotificationResponse, error) {
	attrs := []attribute.KeyValue{
		attribute.String("notification.user_id", userID),
	}
	spanCtx, span := s.tracer.Start(ctx, "notifications.mark_read", trace.WithAttributes(attrs...))
	defer span.End()

	notification, err := s.repo.MarkRead(spanCtx, id, userID)
	if err != nil {
		span.RecordError(err)
		return dto.NotificationResponse{}, err
	}

	return dto.NewNotificationResponse(notification), nil
}

func (s *notificationService) Subscribe(userID string) (<-chan dto.NotificationResponse, func()) {
	channel := make(chan dto.NotificationResponse, notificationBufferSize)

	s.broker.subscribe(userID, channel)
	observability.SSEClientsActive().Inc()

	cleanup := func() {
		s.broker.unsubscribe(userID, channel)
		observability.SSEClientsActive().Dec()
	}

	return channel, cleanup
}

func (s *notificationService) broadcast(notification dto.NotificationResponse) {
	s.broker.broadcast(notification.UserID, notification)
}

func (s *notificationService) publish(ctx context.Context, notification dto.NotificationResponse) error {
	event := notificationEvent{
		Source:       s.nodeID,
		Notification: notification,
		SentAt:       time.Now().UTC(),
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

func (s *notificationService) consumeRedis(ctx context.Context) {
	pubsub := s.redis.Subscribe(ctx, s.redisStream)
	defer func() { _ = pubsub.Close() }()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			s.logger.Error().Err(err).Msg("notification redis subscription closed")
			return
		}
		s.handleEvent([]byte(msg.Payload))
	}
}

func (s *notificationService) consumeNATS(ctx context.Context) {
	sub, err := s.nats.QueueSubscribe(s.natsSubject, "gema-notifications", func(msg *nats.Msg) {
		s.handleEvent(msg.Data)
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to subscribe to nats notifications subject")
		return
	}

	go func() {
		<-ctx.Done()
		if err := sub.Drain(); err != nil {
			s.logger.Warn().Err(err).Msg("failed to drain notification nats subscription")
		}
	}()
}

func (s *notificationService) handleEvent(payload []byte) {
	var event notificationEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		s.logger.Warn().Err(err).Msg("invalid notification event payload")
		return
	}

	if event.Source == s.nodeID {
		return
	}

	notification := event.Notification
	if notification.Type == "" {
		notification.Type = "generic"
	}

	observability.NotificationsPublishedTotal().WithLabelValues(notification.Type).Inc()
	s.broadcast(notification)
}

func (b *notificationBroker) subscribe(userID string, ch chan dto.NotificationResponse) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[userID]; !exists {
		b.subscribers[userID] = make(map[chan dto.NotificationResponse]struct{})
	}
	b.subscribers[userID][ch] = struct{}{}
}

func (b *notificationBroker) unsubscribe(userID string, ch chan dto.NotificationResponse) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subscribers, ok := b.subscribers[userID]; ok {
		delete(subscribers, ch)
		close(ch)
		if len(subscribers) == 0 {
			delete(b.subscribers, userID)
		}
	}
}

func (b *notificationBroker) broadcast(userID string, notification dto.NotificationResponse) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subscribers := b.subscribers[userID]
	for ch := range subscribers {
		select {
		case ch <- notification:
		default:
		}
	}
}
