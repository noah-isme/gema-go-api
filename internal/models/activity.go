package models

import (
	"time"

	"gorm.io/datatypes"
)

// ActivityLog captures auditable events triggered by administrators and teachers.
type ActivityLog struct {
	ID         uint              `gorm:"primaryKey" json:"id"`
	ActorID    uint              `gorm:"not null" json:"actor_id"`
	ActorRole  string            `gorm:"size:32;not null" json:"actor_role"`
	Action     string            `gorm:"size:64;not null" json:"action"`
	EntityType string            `gorm:"size:64;not null" json:"entity_type"`
	EntityID   *uint             `json:"entity_id"`
	Metadata   datatypes.JSONMap `gorm:"type:json" json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
}
