package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Announcement represents a broadcast message displayed to end users.
type Announcement struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Slug      string     `gorm:"size:128;uniqueIndex" json:"slug"`
	Title     string     `gorm:"size:255;not null" json:"title"`
	Body      string     `gorm:"type:text;not null" json:"body"`
	StartsAt  time.Time  `gorm:"index" json:"starts_at"`
	EndsAt    *time.Time `gorm:"index" json:"ends_at"`
	IsPinned  bool       `gorm:"index" json:"is_pinned"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// GalleryItem captures media published in the public gallery.
type GalleryItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Slug      string    `gorm:"size:128;uniqueIndex" json:"slug"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Caption   string    `gorm:"type:text" json:"caption"`
	ImagePath string    `gorm:"size:512;not null" json:"image_path"`
	TagsRaw   string    `gorm:"column:tags;type:text" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Tags      []string  `gorm:"-" json:"tags"`
}

// BeforeSave normalises tag data before persisting.
func (g *GalleryItem) BeforeSave(tx *gorm.DB) error {
	g.TagsRaw = encodeTags(g.Tags)
	return nil
}

// AfterFind hydrates tag list after retrieval.
func (g *GalleryItem) AfterFind(tx *gorm.DB) error {
	g.Tags = decodeTags(g.TagsRaw)
	return nil
}

func encodeTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(strings.ToLower(tag))
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) == 0 {
		return ""
	}
	return "|" + strings.Join(cleaned, "|") + "|"
}

func decodeTags(raw string) []string {
	raw = strings.Trim(raw, "|")
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, "|")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		tags = append(tags, trimmed)
	}
	return tags
}

// ContactSubmission stores inbound enquiries.
type ContactSubmission struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ReferenceID string     `gorm:"size:64;uniqueIndex" json:"reference_id"`
	Name        string     `gorm:"size:128;not null" json:"name"`
	Email       string     `gorm:"size:160;not null" json:"email"`
	Message     string     `gorm:"type:text;not null" json:"message"`
	Source      string     `gorm:"size:64" json:"source"`
	Status      string     `gorm:"size:32;not null" json:"status"`
	Checksum    string     `gorm:"size:128;index" json:"checksum"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
}

// UploadRecord stores metadata about uploaded files.
type UploadRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    *uint     `gorm:"index" json:"user_id"`
	FileName  string    `gorm:"size:255;not null" json:"file_name"`
	URL       string    `gorm:"size:512;not null" json:"url"`
	MimeType  string    `gorm:"size:128;not null" json:"mime_type"`
	SizeBytes int64     `gorm:"not null" json:"size_bytes"`
	Checksum  string    `gorm:"size:128;index" json:"checksum"`
	CreatedAt time.Time `json:"created_at"`
}
