package dto

import "github.com/noah-isme/gema-go-api/internal/models"

// CodingSubmissionRequest represents the payload for creating a submission.
type CodingSubmissionRequest struct {
	TaskID   uint   `json:"task_id" validate:"required,gt=0"`
	Language string `json:"language" validate:"required"`
	Source   string `json:"source" validate:"required,min=1"`
}

// CodingSubmissionResponse represents a coding submission to API consumers.
type CodingSubmissionResponse struct {
	ID          uint                       `json:"id"`
	TaskID      uint                       `json:"task_id"`
	StudentID   uint                       `json:"student_id"`
	Language    string                     `json:"language"`
	Source      string                     `json:"source,omitempty"`
	Status      string                     `json:"status"`
	Output      string                     `json:"output"`
	Error       string                     `json:"error"`
	CPUTimeMs   int64                      `json:"cpu_time_ms"`
	MemoryKB    int64                      `json:"memory_kb"`
	Task        CodingTaskResponse         `json:"task"`
	Evaluations []CodingEvaluationResponse `json:"evaluations"`
}

// CodingEvaluationResponse describes the AI evaluation payload.
type CodingEvaluationResponse struct {
	ID       uint                   `json:"id"`
	Score    float64                `json:"score"`
	Verdict  string                 `json:"verdict"`
	Feedback string                 `json:"feedback"`
	Details  map[string]interface{} `json:"details"`
	Raw      map[string]interface{} `json:"raw"`
	Provider string                 `json:"provider"`
}

// NewCodingSubmissionResponse builds a response DTO from a model.
func NewCodingSubmissionResponse(submission models.CodingSubmission, includeSource bool) CodingSubmissionResponse {
	response := CodingSubmissionResponse{
		ID:        submission.ID,
		TaskID:    submission.TaskID,
		StudentID: submission.StudentID,
		Language:  submission.Language,
		Status:    submission.Status,
		Output:    submission.Output,
		Error:     submission.Error,
		CPUTimeMs: submission.CPUTimeMs,
		MemoryKB:  submission.MemoryKB,
		Task:      NewCodingTaskResponse(submission.Task),
	}

	if includeSource {
		response.Source = submission.Source
	}

	if len(submission.Evaluations) > 0 {
		evals := make([]CodingEvaluationResponse, 0, len(submission.Evaluations))
		for _, evaluation := range submission.Evaluations {
			evals = append(evals, NewCodingEvaluationResponse(evaluation))
		}
		response.Evaluations = evals
	}

	return response
}

// NewCodingEvaluationResponse converts a CodingEvaluation model into a DTO.
func NewCodingEvaluationResponse(evaluation models.CodingEvaluation) CodingEvaluationResponse {
	details := map[string]interface{}(nil)
	if evaluation.Details != nil {
		details = map[string]interface{}(evaluation.Details)
	}
	raw := map[string]interface{}(nil)
	if evaluation.Raw != nil {
		raw = map[string]interface{}(evaluation.Raw)
	}

	return CodingEvaluationResponse{
		ID:       evaluation.ID,
		Score:    evaluation.Score,
		Verdict:  evaluation.Verdict,
		Feedback: evaluation.Feedback,
		Details:  details,
		Raw:      raw,
		Provider: evaluation.Provider,
	}
}
