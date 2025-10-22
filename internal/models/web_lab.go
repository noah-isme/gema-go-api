package models

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// WebAssignment represents a frontend lab assignment definition.
type WebAssignment struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	Title        string          `gorm:"size:255;not null" json:"title"`
	Requirements string          `gorm:"type:text" json:"requirements"`
	Assets       datatypes.JSON  `gorm:"type:json" json:"-"`
	Rubric       string          `gorm:"type:text" json:"rubric"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Submissions  []WebSubmission `gorm:"foreignKey:AssignmentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// SetAssets serializes the provided asset list into the JSON storage column.
func (a *WebAssignment) SetAssets(assets []string) {
	data, err := json.Marshal(assets)
	if err != nil {
		a.Assets = datatypes.JSON([]byte("[]"))
		return
	}
	a.Assets = datatypes.JSON(data)
}

// AssetList deserializes the stored asset bundle into a Go slice.
func (a WebAssignment) AssetList() []string {
	if len(a.Assets) == 0 {
		return nil
	}

	var assets []string
	if err := json.Unmarshal(a.Assets, &assets); err != nil {
		return nil
	}

	return assets
}

// WebSubmission models a student's submission for a web lab assignment.
type WebSubmission struct {
	ID           uint          `gorm:"primaryKey" json:"id"`
	AssignmentID uint          `gorm:"not null" json:"assignment_id"`
	StudentID    uint          `gorm:"not null" json:"student_id"`
	ZipURL       string        `gorm:"size:512" json:"zip_url"`
	Status       string        `gorm:"size:32;not null" json:"status"`
	Feedback     string        `gorm:"type:text" json:"feedback"`
	Score        *float64      `json:"score"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Assignment   WebAssignment `gorm:"foreignKey:AssignmentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"assignment"`
	Student      Student       `gorm:"foreignKey:StudentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"student"`
}

const (
	// WebSubmissionStatusValidated indicates the submission passed automated checks.
	WebSubmissionStatusValidated = "validated"
	// WebSubmissionStatusRejected indicates the submission failed safety validation.
	WebSubmissionStatusRejected = "rejected"
)
