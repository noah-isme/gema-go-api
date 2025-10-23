package dto

import (
	"time"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// WebAssignmentResponse describes the payload returned to API clients.
type WebAssignmentResponse struct {
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	Requirements string    `json:"requirements"`
	Assets       []string  `json:"assets"`
	Rubric       string    `json:"rubric"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewWebAssignmentResponse converts a model into a DTO.
func NewWebAssignmentResponse(model models.WebAssignment) WebAssignmentResponse {
	return WebAssignmentResponse{
		ID:           model.ID,
		Title:        model.Title,
		Requirements: model.Requirements,
		Assets:       model.AssetList(),
		Rubric:       model.Rubric,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

// NewWebAssignmentResponseSlice converts a slice of models into DTOs.
func NewWebAssignmentResponseSlice(assignments []models.WebAssignment) []WebAssignmentResponse {
	responses := make([]WebAssignmentResponse, 0, len(assignments))
	for _, assignment := range assignments {
		responses = append(responses, NewWebAssignmentResponse(assignment))
	}

	return responses
}

// WebSubmissionCreateRequest captures the payload for creating a submission.
type WebSubmissionCreateRequest struct {
	AssignmentID uint `form:"assignment_id" json:"assignment_id" validate:"required"`
	StudentID    uint `form:"student_id" json:"student_id" validate:"required"`
}

// WebSubmissionResponse serializes a submission for API clients.
type WebSubmissionResponse struct {
	ID           uint                   `json:"id"`
	AssignmentID uint                   `json:"assignment_id"`
	StudentID    uint                   `json:"student_id"`
	ZipURL       string                 `json:"zip_url"`
	Status       string                 `json:"status"`
	Feedback     string                 `json:"feedback"`
	Score        *float64               `json:"score"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Assignment   *WebAssignmentResponse `json:"assignment,omitempty"`
}

// NewWebSubmissionResponse converts a submission model into a DTO.
func NewWebSubmissionResponse(model models.WebSubmission) WebSubmissionResponse {
	response := WebSubmissionResponse{
		ID:           model.ID,
		AssignmentID: model.AssignmentID,
		StudentID:    model.StudentID,
		ZipURL:       model.ZipURL,
		Status:       model.Status,
		Feedback:     model.Feedback,
		Score:        model.Score,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}

	if model.Assignment.ID != 0 {
		assignment := NewWebAssignmentResponse(model.Assignment)
		response.Assignment = &assignment
	}

	return response
}
