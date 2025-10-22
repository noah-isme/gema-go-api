package dto

import (
	"time"

	"github.com/noah-isme/gema-go-api/internal/models"
)

const isoLayout = time.RFC3339

// AssignmentCreateRequest describes the payload for creating a new assignment.
type AssignmentCreateRequest struct {
	Title       string `form:"title" json:"title" validate:"required,min=3"`
	Description string `form:"description" json:"description" validate:"required,min=10"`
	DueDate     string `form:"due_date" json:"due_date" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
}

// AssignmentUpdateRequest describes the payload for updating an assignment.
type AssignmentUpdateRequest struct {
	Title       *string `form:"title" json:"title" validate:"omitempty,min=3"`
	Description *string `form:"description" json:"description" validate:"omitempty,min=10"`
	DueDate     *string `form:"due_date" json:"due_date" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// AssignmentResponse is the serialized representation returned to API clients.
type AssignmentResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	FileURL     string    `json:"file_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewAssignmentResponse converts a model into a DTO.
func NewAssignmentResponse(model models.Assignment) AssignmentResponse {
	return AssignmentResponse{
		ID:          model.ID,
		Title:       model.Title,
		Description: model.Description,
		DueDate:     model.DueDate,
		FileURL:     model.FileURL,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

// NewAssignmentResponseSlice converts a slice of models into DTOs.
func NewAssignmentResponseSlice(assignments []models.Assignment) []AssignmentResponse {
	responses := make([]AssignmentResponse, 0, len(assignments))
	for _, assignment := range assignments {
		responses = append(responses, NewAssignmentResponse(assignment))
	}

	return responses
}
