package models

import (
	"strings"
	"time"
)

// CodingTask represents a coding lab exercise available to students.
type CodingTask struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Title          string    `gorm:"size:255;not null" json:"title"`
	Prompt         string    `gorm:"type:text;not null" json:"prompt"`
	StarterCode    string    `gorm:"type:text" json:"starter_code"`
	Language       string    `gorm:"size:32;not null" json:"language"`
	Difficulty     string    `gorm:"size:32;not null" json:"difficulty"`
	Tags           string    `gorm:"type:text" json:"tags"`
	ExpectedOutput string    `gorm:"type:text" json:"expected_output"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TagsSlice returns the tags as a slice of strings.
func (t CodingTask) TagsSlice() []string {
	if t.Tags == "" {
		return nil
	}

	parts := strings.Split(t.Tags, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}
