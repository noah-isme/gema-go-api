package service

import (
	"context"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
)

type stubDiscussionRepo struct {
	thread  models.DiscussionThread
	replies []models.DiscussionReply
}

func (s *stubDiscussionRepo) ListThreads(ctx context.Context, limit, offset int) ([]models.DiscussionThread, error) {
	return []models.DiscussionThread{s.thread}, nil
}

func (s *stubDiscussionRepo) GetThread(ctx context.Context, id uint) (models.DiscussionThread, error) {
	return s.thread, nil
}

func (s *stubDiscussionRepo) GetThreadWithReplies(ctx context.Context, id uint) (models.DiscussionThread, error) {
	thread := s.thread
	thread.Replies = s.replies
	return thread, nil
}

func (s *stubDiscussionRepo) CreateThread(ctx context.Context, thread *models.DiscussionThread) error {
	s.thread = *thread
	return nil
}

func (s *stubDiscussionRepo) UpdateThread(ctx context.Context, thread *models.DiscussionThread) error {
	s.thread = *thread
	return nil
}

func (s *stubDiscussionRepo) DeleteThread(ctx context.Context, id uint) error {
	return nil
}

func (s *stubDiscussionRepo) CreateReply(ctx context.Context, reply *models.DiscussionReply) error {
	reply.ID = uint(len(s.replies) + 1)
	s.replies = append(s.replies, *reply)
	return nil
}

func (s *stubDiscussionRepo) ListReplies(ctx context.Context, threadID uint, limit, offset int) ([]models.DiscussionReply, error) {
	return s.replies, nil
}

type stubNotificationPublisher struct {
	calls []dto.NotificationCreateRequest
}

func (s *stubNotificationPublisher) Publish(ctx context.Context, payload dto.NotificationCreateRequest) (dto.NotificationResponse, error) {
	s.calls = append(s.calls, payload)
	return dto.NotificationResponse{UserID: payload.UserID, Type: payload.Type, Message: payload.Message}, nil
}

func TestDiscussionServiceCreateReplySendsNotifications(t *testing.T) {
	repo := &stubDiscussionRepo{thread: models.DiscussionThread{ID: 1, Title: "Weekly Standup", AuthorID: "42"}}
	notifications := &stubNotificationPublisher{}
	svc := NewDiscussionService(repo, notifications, validator.New(validator.WithRequiredStructEnabled()), zerolog.Nop())

	reply, err := svc.CreateReply(context.Background(), "24", "student", dto.DiscussionReplyCreateRequest{
		ThreadID: 1,
		Content:  "<script>alert(1)</script>Hello @42 and @99",
	})
	require.NoError(t, err)
	require.Equal(t, "Hello @42 and @99", reply.Content)
	require.Len(t, notifications.calls, 2)
	users := []string{notifications.calls[0].UserID, notifications.calls[1].UserID}
	require.ElementsMatch(t, []string{"42", "99"}, users)
	for _, call := range notifications.calls {
		require.Equal(t, "discussion_reply", call.Type)
	}
}
