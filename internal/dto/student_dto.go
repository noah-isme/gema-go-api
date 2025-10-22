package dto

import "time"

// StudentDashboardResponse aggregates tutorial progress for a student.
type StudentDashboardResponse struct {
	Summary           ProgressSummary      `json:"summary"`
	Pending           []AssignmentProgress `json:"pending_assignments"`
	RecentSubmissions []SubmissionActivity `json:"recent_submissions"`
}

// ProgressSummary captures aggregated statistics for the dashboard.
type ProgressSummary struct {
	TotalAssignments int     `json:"total_assignments"`
	Submitted        int     `json:"submitted"`
	Graded           int     `json:"graded"`
	Pending          int     `json:"pending"`
	Overdue          int     `json:"overdue"`
	AverageGrade     float64 `json:"average_grade"`
	CompletionRate   float64 `json:"completion_rate"`
}

// AssignmentProgress describes the state of a single assignment relative to a student.
type AssignmentProgress struct {
	AssignmentID  uint      `json:"assignment_id"`
	Title         string    `json:"title"`
	DueDate       time.Time `json:"due_date"`
	FileURL       string    `json:"file_url"`
	Status        string    `json:"status"`
	SubmissionID  *uint     `json:"submission_id"`
	SubmissionURL string    `json:"submission_url"`
	Grade         *float64  `json:"grade"`
	Feedback      string    `json:"feedback"`
	UpdatedAt     time.Time `json:"updated_at"`
	Overdue       bool      `json:"overdue"`
}

// SubmissionActivity details recent submission events for activity feed.
type SubmissionActivity struct {
	SubmissionID   uint      `json:"submission_id"`
	AssignmentID   uint      `json:"assignment_id"`
	AssignmentName string    `json:"assignment_name"`
	Status         string    `json:"status"`
	Grade          *float64  `json:"grade"`
	Feedback       string    `json:"feedback"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
