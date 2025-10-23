package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/datatypes"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// ErrDiscussionForbidden indicates the user attempted an operation they are not allowed to perform.
var ErrDiscussionForbidden = errors.New("insufficient permissions for discussion operation")

// NotificationPublisher exposes the subset of notification service needed by discussions.
type NotificationPublisher interface {
	Publish(ctx context.Context, payload dto.NotificationCreateRequest) (dto.NotificationResponse, error)
}

// DiscussionService exposes discussion thread use-cases.
type DiscussionService interface {
	ListThreads(ctx context.Context, limit, offset int) ([]dto.DiscussionThreadResponse, error)
	GetThread(ctx context.Context, id uint, includeReplies bool) (dto.DiscussionThreadResponse, error)
	CreateThread(ctx context.Context, authorID, role string, payload dto.DiscussionThreadCreateRequest) (dto.DiscussionThreadResponse, error)
	UpdateThread(ctx context.Context, id uint, authorID, role string, payload dto.DiscussionThreadUpdateRequest) (dto.DiscussionThreadResponse, error)
	DeleteThread(ctx context.Context, id uint, authorID, role string) error
	ListReplies(ctx context.Context, threadID uint, limit, offset int) ([]dto.DiscussionReplyResponse, error)
	CreateReply(ctx context.Context, authorID, role string, payload dto.DiscussionReplyCreateRequest) (dto.DiscussionReplyResponse, error)
}

type discussionService struct {
	repo           repository.DiscussionRepository
	notifications  NotificationPublisher
	validator      *validator.Validate
	logger         zerolog.Logger
	tracer         trace.Tracer
	sanitizer      *bluemonday.Policy
	mentionPattern *regexp.Regexp
	now            func() time.Time
}

// NewDiscussionService constructs a discussion service.
func NewDiscussionService(repo repository.DiscussionRepository, notifications NotificationPublisher, validate *validator.Validate, logger zerolog.Logger) DiscussionService {
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("br")

	return &discussionService{
		repo:           repo,
		notifications:  notifications,
		validator:      validate,
		logger:         logger.With().Str("component", "discussion_service").Logger(),
		tracer:         otel.Tracer("github.com/noah-isme/gema-go-api/internal/service/discussion"),
		sanitizer:      policy,
		mentionPattern: regexp.MustCompile(`@([a-zA-Z0-9_\-:]+)`),
		now:            time.Now,
	}
}

func (s *discussionService) ListThreads(ctx context.Context, limit, offset int) ([]dto.DiscussionThreadResponse, error) {
	threads, err := s.repo.ListThreads(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return dto.NewDiscussionThreadResponseSlice(threads), nil
}

func (s *discussionService) GetThread(ctx context.Context, id uint, includeReplies bool) (dto.DiscussionThreadResponse, error) {
	var (
		thread models.DiscussionThread
		err    error
	)

	if includeReplies {
		thread, err = s.repo.GetThreadWithReplies(ctx, id)
	} else {
		thread, err = s.repo.GetThread(ctx, id)
	}
	if err != nil {
		return dto.DiscussionThreadResponse{}, err
	}

	return dto.NewDiscussionThreadResponse(thread), nil
}

func (s *discussionService) CreateThread(ctx context.Context, authorID, role string, payload dto.DiscussionThreadCreateRequest) (dto.DiscussionThreadResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.DiscussionThreadResponse{}, err
	}

	sanitizedTitle := strings.TrimSpace(s.sanitizer.Sanitize(payload.Title))
	if sanitizedTitle == "" {
		return dto.DiscussionThreadResponse{}, errors.New("thread title empty after sanitization")
	}

	attrs := []attribute.KeyValue{
		attribute.String("discussion.author_id", authorID),
		attribute.String("discussion.role", role),
	}

	spanCtx, span := s.tracer.Start(ctx, "discussion.create", trace.WithAttributes(attrs...))
	defer span.End()

	thread := models.DiscussionThread{
		Title:    sanitizedTitle,
		AuthorID: authorID,
		Metadata: datatypes.JSONMap{"created_by_role": role},
	}

	if err := s.repo.CreateThread(spanCtx, &thread); err != nil {
		span.RecordError(err)
		return dto.DiscussionThreadResponse{}, err
	}

	s.logger.Info().Uint("thread_id", thread.ID).Str("author_id", authorID).Msg("discussion thread created")

	return dto.NewDiscussionThreadResponse(thread), nil
}

func (s *discussionService) UpdateThread(ctx context.Context, id uint, authorID, role string, payload dto.DiscussionThreadUpdateRequest) (dto.DiscussionThreadResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.DiscussionThreadResponse{}, err
	}

	thread, err := s.repo.GetThread(ctx, id)
	if err != nil {
		return dto.DiscussionThreadResponse{}, err
	}

	if err := s.authorizeMutation(thread.AuthorID, authorID, role); err != nil {
		return dto.DiscussionThreadResponse{}, err
	}

	if payload.Title != nil {
		sanitized := strings.TrimSpace(s.sanitizer.Sanitize(*payload.Title))
		if sanitized == "" {
			return dto.DiscussionThreadResponse{}, errors.New("thread title empty after sanitization")
		}
		thread.Title = sanitized
	}

	if err := s.repo.UpdateThread(ctx, &thread); err != nil {
		return dto.DiscussionThreadResponse{}, err
	}

	return dto.NewDiscussionThreadResponse(thread), nil
}

func (s *discussionService) DeleteThread(ctx context.Context, id uint, authorID, role string) error {
	thread, err := s.repo.GetThread(ctx, id)
	if err != nil {
		return err
	}

	if err := s.authorizeMutation(thread.AuthorID, authorID, role); err != nil {
		return err
	}

	return s.repo.DeleteThread(ctx, id)
}

func (s *discussionService) ListReplies(ctx context.Context, threadID uint, limit, offset int) ([]dto.DiscussionReplyResponse, error) {
	replies, err := s.repo.ListReplies(ctx, threadID, limit, offset)
	if err != nil {
		return nil, err
	}
	return dto.NewDiscussionReplyResponseSlice(replies), nil
}

func (s *discussionService) CreateReply(ctx context.Context, authorID, role string, payload dto.DiscussionReplyCreateRequest) (dto.DiscussionReplyResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.DiscussionReplyResponse{}, err
	}

	sanitized := strings.TrimSpace(s.sanitizer.Sanitize(payload.Content))
	if sanitized == "" {
		return dto.DiscussionReplyResponse{}, errors.New("reply content empty after sanitization")
	}

	thread, err := s.repo.GetThread(ctx, payload.ThreadID)
	if err != nil {
		return dto.DiscussionReplyResponse{}, err
	}

	reply := models.DiscussionReply{
		ThreadID: payload.ThreadID,
		AuthorID: authorID,
		Content:  sanitized,
	}

	if err := s.repo.CreateReply(ctx, &reply); err != nil {
		return dto.DiscussionReplyResponse{}, err
	}

	s.dispatchNotifications(ctx, thread, reply)

	return dto.NewDiscussionReplyResponse(reply), nil
}

func (s *discussionService) authorizeMutation(ownerID, actorID, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if actorID == ownerID {
		return nil
	}
	if role == "admin" || role == "teacher" {
		return nil
	}
	return ErrDiscussionForbidden
}

func (s *discussionService) dispatchNotifications(ctx context.Context, thread models.DiscussionThread, reply models.DiscussionReply) {
	if s.notifications == nil {
		return
	}

	mentions := s.extractMentions(reply.Content)

	targets := make(map[string]struct{})
	if thread.AuthorID != "" && thread.AuthorID != reply.AuthorID {
		targets[thread.AuthorID] = struct{}{}
	}
	for _, mention := range mentions {
		if mention == reply.AuthorID {
			continue
		}
		targets[mention] = struct{}{}
	}

	for userID := range targets {
		message := fmt.Sprintf("New reply in thread '%s'", thread.Title)
		payload := dto.NotificationCreateRequest{
			UserID:  userID,
			Type:    "discussion_reply",
			Message: message,
		}
		if _, err := s.notifications.Publish(ctx, payload); err != nil {
			s.logger.Warn().Err(err).Str("user_id", userID).Msg("failed to publish discussion notification")
		}
	}
}

func (s *discussionService) extractMentions(content string) []string {
	matches := s.mentionPattern.FindAllStringSubmatch(content, -1)
	mentions := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		mention := strings.TrimSpace(match[1])
		if mention != "" {
			mentions = append(mentions, mention)
		}
	}
	return mentions
}
