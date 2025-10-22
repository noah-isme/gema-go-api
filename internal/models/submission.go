package models

import "time"

// Submission represents a file submitted by a student for an assignment.
type Submission struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	AssignmentID uint       `gorm:"not null" json:"assignment_id"`
	StudentID    uint       `gorm:"not null" json:"student_id"`
	FileURL      string     `gorm:"size:512" json:"file_url"`
	Status       string     `gorm:"size:32;not null" json:"status"`
	Grade        *float64   `json:"grade"`
	Feedback     string     `gorm:"type:text" json:"feedback"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Assignment   Assignment `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"assignment"`
	Student      Student    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"student"`
}

const (
	// SubmissionStatusSubmitted indicates the submission has been uploaded but not graded.
	SubmissionStatusSubmitted = "submitted"
	// SubmissionStatusGraded indicates the submission has been evaluated.
	SubmissionStatusGraded = "graded"
)

// IsGraded reports whether the submission has a final grade.
func (s Submission) IsGraded() bool {
	return s.Status == SubmissionStatusGraded
}
