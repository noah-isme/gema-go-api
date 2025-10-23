package models

import (
	"time"

	"gorm.io/datatypes"
)

// ChatMessage represents a single chat payload exchanged between users or rooms.
type ChatMessage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SenderID   string    `gorm:"size:64;index" json:"sender_id"`
	ReceiverID string    `gorm:"size:64;index" json:"receiver_id"`
	RoomID     string    `gorm:"size:128;index" json:"room_id"`
	Content    string    `gorm:"type:text" json:"content"`
	Type       string    `gorm:"size:32;default:text" json:"type"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Notification represents a push notification targeted to a specific user.
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"size:64;index" json:"user_id"`
	Type      string    `gorm:"size:64" json:"type"`
	Message   string    `gorm:"type:text" json:"message"`
	Read      bool      `gorm:"not null;default:false" json:"read"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DiscussionThread represents a discussion forum topic.
type DiscussionThread struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Title     string            `gorm:"size:255;not null" json:"title"`
	AuthorID  string            `gorm:"size:64;index" json:"author_id"`
	Metadata  datatypes.JSONMap `gorm:"type:json" json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Replies   []DiscussionReply `json:"replies"`
}

// DiscussionReply represents a reply within a discussion thread.
type DiscussionReply struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ThreadID  uint      `gorm:"index;not null" json:"thread_id"`
	AuthorID  string    `gorm:"size:64;index" json:"author_id"`
	Content   string    `gorm:"type:text" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
