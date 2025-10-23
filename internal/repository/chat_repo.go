package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// ChatRepository persists chat messages for history and compliance needs.
type ChatRepository interface {
	Save(ctx context.Context, message *models.ChatMessage) error
	ListByRoom(ctx context.Context, roomID string, before time.Time, limit int) ([]models.ChatMessage, error)
	ListBySender(ctx context.Context, senderID string, limit int) ([]models.ChatMessage, error)
	LatestByRoom(ctx context.Context, roomID string) (models.ChatMessage, error)
}

type chatRepository struct {
	db *gorm.DB
}

// NewChatRepository constructs a chat repository backed by GORM.
func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) Save(ctx context.Context, message *models.ChatMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *chatRepository) ListByRoom(ctx context.Context, roomID string, before time.Time, limit int) ([]models.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := r.db.WithContext(ctx).Where("room_id = ?", roomID)
	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}

	var messages []models.ChatMessage
	if err := query.Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}

	// Reverse to chronological order ascending for clients.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *chatRepository) ListBySender(ctx context.Context, senderID string, limit int) ([]models.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var messages []models.ChatMessage
	if err := r.db.WithContext(ctx).Where("sender_id = ?", senderID).Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *chatRepository) LatestByRoom(ctx context.Context, roomID string) (models.ChatMessage, error) {
	var message models.ChatMessage
	err := r.db.WithContext(ctx).Where("room_id = ?", roomID).Order("created_at DESC").First(&message).Error
	if err != nil {
		return models.ChatMessage{}, err
	}
	return message, nil
}
