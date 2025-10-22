package models

import (
	"time"

	"gorm.io/datatypes"
)

// CodingEvaluation captures the outcome of an AI evaluation for a submission.
type CodingEvaluation struct {
	ID           uint              `gorm:"primaryKey" json:"id"`
	SubmissionID uint              `gorm:"not null" json:"submission_id"`
	Score        float64           `gorm:"not null" json:"score"`
	Verdict      string            `gorm:"size:64" json:"verdict"`
	Feedback     string            `gorm:"type:text" json:"feedback"`
	Details      datatypes.JSONMap `json:"details"`
	Raw          datatypes.JSONMap `json:"raw"`
	Provider     string            `gorm:"size:32" json:"provider"`
	CreatedAt    time.Time         `json:"created_at"`
	Submission   CodingSubmission  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"submission"`
}
