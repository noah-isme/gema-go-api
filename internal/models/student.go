package models

import "time"

// Student represents a learner that can submit assignments.
type Student struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Email     string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
