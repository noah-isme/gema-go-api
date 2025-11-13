package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RoadmapStage represents a learning stage surfaced on the student dashboard.
type RoadmapStage struct {
	ID             uint              `gorm:"primaryKey"`
	Slug           string            `gorm:"size:160;uniqueIndex"`
	Title          string            `gorm:"size:255;not null"`
	Description    string            `gorm:"type:text"`
	Sequence       int               `gorm:"index"`
	EstimatedHours int               `gorm:"default:2"`
	Icon           string            `gorm:"size:64"`
	TagsRaw        string            `gorm:"column:tags;type:text"`
	Skills         datatypes.JSONMap `gorm:"type:json"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Tags           []string          `gorm:"-"`
}

// BeforeSave normalises roadmap stage tags.
func (r *RoadmapStage) BeforeSave(tx *gorm.DB) error {
	r.TagsRaw = encodeTags(r.Tags)
	if r.Sequence < 0 {
		r.Sequence = 0
	}
	if r.EstimatedHours <= 0 {
		r.EstimatedHours = 2
	}
	return nil
}

// AfterFind hydrates tags after loading from DB.
func (r *RoadmapStage) AfterFind(tx *gorm.DB) error {
	r.Tags = decodeTags(r.TagsRaw)
	return nil
}
