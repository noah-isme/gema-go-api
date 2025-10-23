package models

import (
	"time"

	"gorm.io/datatypes"
)

// Assignment represents a tutorial assignment definition.
type Assignment struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Title       string            `gorm:"size:255;not null" json:"title"`
	Description string            `gorm:"type:text" json:"description"`
	DueDate     time.Time         `gorm:"not null" json:"due_date"`
	FileURL     string            `gorm:"size:512" json:"file_url"`
	MaxScore    float64           `gorm:"not null;default:100" json:"max_score"`
	Rubric      datatypes.JSONMap `gorm:"type:json" json:"rubric"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Submissions []Submission
}

// IsPastDue returns true when the assignment deadline has already passed.
func (a Assignment) IsPastDue(reference time.Time) bool {
	return reference.After(a.DueDate)
}
