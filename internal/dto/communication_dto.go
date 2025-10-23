package dto

import (
	"time"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// ChatSendRequest represents the payload sent from clients to broadcast a chat message.
type ChatSendRequest struct {
	RoomID     string `json:"room_id" validate:"required,min=3,max=128"`
	Content    string `json:"content" validate:"required,min=1,max=4000"`
	Type       string `json:"type" validate:"omitempty,oneof=text image file system"`
	ReceiverID string `json:"receiver_id" validate:"omitempty,max=64"`
}

// ChatHistoryQuery represents query filters for retrieving chat history.
type ChatHistoryQuery struct {
	RoomID string     `query:"room_id" validate:"required,min=3,max=128"`
	Before *time.Time `query:"before"`
	Limit  int        `query:"limit" validate:"omitempty,min=1,max=100"`
}

// ChatMessageResponse is the serialized representation of a chat message.
type ChatMessageResponse struct {
	ID         uint      `json:"id"`
	RoomID     string    `json:"room_id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id,omitempty"`
	Content    string    `json:"content"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewChatMessageResponse converts a model into a DTO.
func NewChatMessageResponse(message models.ChatMessage) ChatMessageResponse {
	return ChatMessageResponse{
		ID:         message.ID,
		RoomID:     message.RoomID,
		SenderID:   message.SenderID,
		ReceiverID: message.ReceiverID,
		Content:    message.Content,
		Type:       message.Type,
		CreatedAt:  message.CreatedAt,
	}
}

// NewChatMessageResponseSlice converts a slice of models into DTOs.
func NewChatMessageResponseSlice(messages []models.ChatMessage) []ChatMessageResponse {
	out := make([]ChatMessageResponse, 0, len(messages))
	for _, message := range messages {
		out = append(out, NewChatMessageResponse(message))
	}
	return out
}

// NotificationCreateRequest describes the payload to create a notification.
type NotificationCreateRequest struct {
	UserID  string `json:"user_id" validate:"required,max=64"`
	Type    string `json:"type" validate:"required,max=64"`
	Message string `json:"message" validate:"required,min=1,max=2000"`
}

// NotificationResponse represents notification data returned to clients.
type NotificationResponse struct {
	ID        uint      `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewNotificationResponse converts a notification model to DTO.
func NewNotificationResponse(model models.Notification) NotificationResponse {
	return NotificationResponse{
		ID:        model.ID,
		UserID:    model.UserID,
		Type:      model.Type,
		Message:   model.Message,
		Read:      model.Read,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

// NewNotificationResponseSlice converts a slice to DTOs.
func NewNotificationResponseSlice(items []models.Notification) []NotificationResponse {
	out := make([]NotificationResponse, 0, len(items))
	for _, item := range items {
		out = append(out, NewNotificationResponse(item))
	}
	return out
}

// DiscussionThreadCreateRequest is the payload to create a thread.
type DiscussionThreadCreateRequest struct {
	Title string `json:"title" validate:"required,min=3,max=255"`
}

// DiscussionThreadUpdateRequest updates an existing thread.
type DiscussionThreadUpdateRequest struct {
	Title *string `json:"title" validate:"omitempty,min=3,max=255"`
}

// DiscussionThreadResponse describes a thread returned by the API.
type DiscussionThreadResponse struct {
	ID        uint                      `json:"id"`
	Title     string                    `json:"title"`
	AuthorID  string                    `json:"author_id"`
	Metadata  map[string]string         `json:"metadata,omitempty"`
	CreatedAt time.Time                 `json:"created_at"`
	UpdatedAt time.Time                 `json:"updated_at"`
	Replies   []DiscussionReplyResponse `json:"replies,omitempty"`
}

// DiscussionReplyCreateRequest creates a reply on a thread.
type DiscussionReplyCreateRequest struct {
	ThreadID uint   `json:"thread_id" validate:"required"`
	Content  string `json:"content" validate:"required,min=1,max=5000"`
}

// DiscussionReplyResponse describes a serialized reply.
type DiscussionReplyResponse struct {
	ID        uint      `json:"id"`
	ThreadID  uint      `json:"thread_id"`
	AuthorID  string    `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewDiscussionThreadResponse converts a model into a DTO including replies when preloaded.
func NewDiscussionThreadResponse(model models.DiscussionThread) DiscussionThreadResponse {
	response := DiscussionThreadResponse{
		ID:        model.ID,
		Title:     model.Title,
		AuthorID:  model.AuthorID,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
	if model.Metadata != nil {
		response.Metadata = make(map[string]string)
		for key, value := range model.Metadata {
			if str, ok := value.(string); ok {
				response.Metadata[key] = str
			}
		}
	}
	if len(model.Replies) > 0 {
		replies := make([]DiscussionReplyResponse, 0, len(model.Replies))
		for _, reply := range model.Replies {
			replies = append(replies, NewDiscussionReplyResponse(reply))
		}
		response.Replies = replies
	}
	return response
}

// NewDiscussionThreadResponseSlice converts slice of threads to DTOs.
func NewDiscussionThreadResponseSlice(items []models.DiscussionThread) []DiscussionThreadResponse {
	out := make([]DiscussionThreadResponse, 0, len(items))
	for _, item := range items {
		out = append(out, NewDiscussionThreadResponse(item))
	}
	return out
}

// NewDiscussionReplyResponse converts reply model to DTO.
func NewDiscussionReplyResponse(model models.DiscussionReply) DiscussionReplyResponse {
	return DiscussionReplyResponse{
		ID:        model.ID,
		ThreadID:  model.ThreadID,
		AuthorID:  model.AuthorID,
		Content:   model.Content,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

// NewDiscussionReplyResponseSlice converts replies to DTOs.
func NewDiscussionReplyResponseSlice(items []models.DiscussionReply) []DiscussionReplyResponse {
	out := make([]DiscussionReplyResponse, 0, len(items))
	for _, item := range items {
		out = append(out, NewDiscussionReplyResponse(item))
	}
	return out
}
