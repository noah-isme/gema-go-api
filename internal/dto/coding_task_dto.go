package dto

import "github.com/noah-isme/gema-go-api/internal/models"

// CodingTaskFilter defines query parameters for listing coding tasks.
type CodingTaskFilter struct {
	Language   string   `query:"language"`
	Difficulty string   `query:"difficulty"`
	Tags       []string `query:"tags"`
	Search     string   `query:"search"`
	Page       int      `query:"page"`
	PageSize   int      `query:"page_size"`
}

// Pagination describes pagination metadata for list responses.
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
}

// CodingTaskResponse represents a coding task returned by the API.
type CodingTaskResponse struct {
	ID             uint     `json:"id"`
	Title          string   `json:"title"`
	Prompt         string   `json:"prompt"`
	StarterCode    string   `json:"starter_code"`
	Language       string   `json:"language"`
	Difficulty     string   `json:"difficulty"`
	Tags           []string `json:"tags"`
	ExpectedOutput string   `json:"expected_output"`
}

// CodingTaskListResponse wraps coding tasks and pagination metadata.
type CodingTaskListResponse struct {
	Items      []CodingTaskResponse `json:"items"`
	Pagination Pagination           `json:"pagination"`
}

// CodingTaskDetailResponse extends CodingTaskResponse with metadata.
type CodingTaskDetailResponse struct {
	CodingTaskResponse
}

// NewCodingTaskResponse builds a response DTO from the model.
func NewCodingTaskResponse(task models.CodingTask) CodingTaskResponse {
	return CodingTaskResponse{
		ID:             task.ID,
		Title:          task.Title,
		Prompt:         task.Prompt,
		StarterCode:    task.StarterCode,
		Language:       task.Language,
		Difficulty:     task.Difficulty,
		Tags:           task.TagsSlice(),
		ExpectedOutput: task.ExpectedOutput,
	}
}

// NewCodingTaskDetail builds a detail DTO from the model.
func NewCodingTaskDetail(task models.CodingTask) CodingTaskDetailResponse {
	return CodingTaskDetailResponse{CodingTaskResponse: NewCodingTaskResponse(task)}
}

// NewCodingTaskListResponse builds a list response from models and pagination meta.
func NewCodingTaskListResponse(tasks []models.CodingTask, pagination Pagination) CodingTaskListResponse {
	items := make([]CodingTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, NewCodingTaskResponse(task))
	}

	return CodingTaskListResponse{
		Items:      items,
		Pagination: pagination,
	}
}
