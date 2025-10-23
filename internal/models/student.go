package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	// StudentStatusActive marks a student as active and in good standing.
	StudentStatusActive = "active"
	// StudentStatusInactive marks a student as temporarily inactive.
	StudentStatusInactive = "inactive"
	// StudentStatusArchived marks a student that has been soft deleted.
	StudentStatusArchived = "archived"
	// StudentStatusSuspended marks a student that has been suspended pending review.
	StudentStatusSuspended = "suspended"
)

// Student represents a learner that can submit assignments.
type Student struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Name      string            `gorm:"size:255;not null" json:"name"`
	Email     string            `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Class     string            `gorm:"size:128" json:"class"`
	Status    string            `gorm:"size:32;not null;default:active" json:"status"`
	Flagged   bool              `gorm:"not null;default:false" json:"flagged"`
	Notes     string            `gorm:"type:text" json:"notes"`
	Flags     datatypes.JSONMap `gorm:"type:json" json:"flags"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	DeletedAt gorm.DeletedAt    `gorm:"index" json:"-"`
}

// IsActive returns true when the student is active and not soft deleted.
func (s Student) IsActive() bool {
	return s.Status == StudentStatusActive && s.DeletedAt.Valid == false
}
