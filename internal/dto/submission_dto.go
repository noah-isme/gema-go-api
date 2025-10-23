package dto

import (
	"time"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// SubmissionCreateRequest describes the multipart payload for submission upload.
type SubmissionCreateRequest struct {
	AssignmentID uint `form:"assignment_id" validate:"required,gt=0"`
	StudentID    uint `form:"student_id" validate:"required,gt=0"`
}

// SubmissionUpdateRequest is used to grade or update a submission.
type SubmissionUpdateRequest struct {
	Status   *string  `json:"status" validate:"omitempty,oneof=submitted graded"`
	Grade    *float64 `json:"grade" validate:"omitempty,gte=0,lte=100"`
	Feedback *string  `json:"feedback" validate:"omitempty,min=3"`
}

// SubmissionFilter describes query string filters for listing submissions.
type SubmissionFilter struct {
	AssignmentID *uint   `query:"assignment_id"`
	StudentID    *uint   `query:"student_id"`
	Status       *string `query:"status" validate:"omitempty,oneof=submitted graded"`
}

// SubmissionResponse is returned to API clients when viewing submissions.
type SubmissionResponse struct {
	ID           uint                             `json:"id"`
	AssignmentID uint                             `json:"assignment_id"`
	StudentID    uint                             `json:"student_id"`
	FileURL      string                           `json:"file_url"`
	Status       string                           `json:"status"`
	Grade        *float64                         `json:"grade"`
	Feedback     string                           `json:"feedback"`
	GradedBy     *uint                            `json:"graded_by"`
	GradedAt     *time.Time                       `json:"graded_at"`
	History      []SubmissionGradeHistoryResponse `json:"history"`
	CreatedAt    time.Time                        `json:"created_at"`
	UpdatedAt    time.Time                        `json:"updated_at"`
	Assignment   AssignmentLite                   `json:"assignment"`
	Student      StudentLite                      `json:"student"`
}

// AssignmentLite summarizes an assignment in submission responses.
type AssignmentLite struct {
	ID      uint      `json:"id"`
	Title   string    `json:"title"`
	DueDate time.Time `json:"due_date"`
}

// SubmissionGradeHistoryResponse serializes grading history entries.
type SubmissionGradeHistoryResponse struct {
	Score    float64   `json:"score"`
	Feedback string    `json:"feedback"`
	GradedBy uint      `json:"graded_by"`
	GradedAt time.Time `json:"graded_at"`
}

// StudentLite summarizes a student without exposing full profile data.
type StudentLite struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// NewSubmissionResponse converts a Submission model into a DTO.
func NewSubmissionResponse(model models.Submission) SubmissionResponse {
	response := SubmissionResponse{
		ID:           model.ID,
		AssignmentID: model.AssignmentID,
		StudentID:    model.StudentID,
		FileURL:      model.FileURL,
		Status:       model.Status,
		Grade:        model.Grade,
		Feedback:     model.Feedback,
		GradedBy:     model.GradedBy,
		GradedAt:     model.GradedAt,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}

	if model.Assignment.ID != 0 {
		response.Assignment = AssignmentLite{
			ID:      model.Assignment.ID,
			Title:   model.Assignment.Title,
			DueDate: model.Assignment.DueDate,
		}
	}

	if model.Student.ID != 0 {
		response.Student = StudentLite{
			ID:    model.Student.ID,
			Name:  model.Student.Name,
			Email: model.Student.Email,
		}
	}

	if len(model.History) > 0 {
		history := make([]SubmissionGradeHistoryResponse, 0, len(model.History))
		for _, entry := range model.History {
			history = append(history, SubmissionGradeHistoryResponse{
				Score:    entry.Score,
				Feedback: entry.Feedback,
				GradedBy: entry.GradedBy,
				GradedAt: entry.GradedAt,
			})
		}
		response.History = history
	}

	return response
}

// NewSubmissionResponseSlice converts submission models into DTOs.
func NewSubmissionResponseSlice(models []models.Submission) []SubmissionResponse {
	responses := make([]SubmissionResponse, 0, len(models))
	for _, submission := range models {
		responses = append(responses, NewSubmissionResponse(submission))
	}

	return responses
}
