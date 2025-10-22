package models

import "time"

// CodingSubmissionStatus enumerates possible submission states.
const (
	CodingSubmissionStatusPending   = "pending"
	CodingSubmissionStatusCompleted = "completed"
	CodingSubmissionStatusFailed    = "failed"
	CodingSubmissionStatusTimeout   = "timeout"
	CodingSubmissionStatusEvaluated = "evaluated"
)

// CodingSubmission represents a student's code submission for a coding task.
type CodingSubmission struct {
	ID          uint               `gorm:"primaryKey" json:"id"`
	TaskID      uint               `gorm:"not null" json:"task_id"`
	StudentID   uint               `gorm:"not null" json:"student_id"`
	Language    string             `gorm:"size:32;not null" json:"language"`
	Source      string             `gorm:"type:text" json:"source"`
	Status      string             `gorm:"size:32;not null" json:"status"`
	Output      string             `gorm:"type:text" json:"output"`
	Error       string             `gorm:"type:text" json:"error"`
	CPUTimeMs   int64              `gorm:"default:0" json:"cpu_time_ms"`
	MemoryKB    int64              `gorm:"default:0" json:"memory_kb"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Task        CodingTask         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Evaluations []CodingEvaluation `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// HasBeenEvaluated reports whether the submission has evaluation feedback.
func (s CodingSubmission) HasBeenEvaluated() bool {
	return s.Status == CodingSubmissionStatusEvaluated
}
