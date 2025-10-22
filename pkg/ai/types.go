package ai

import "context"

// EvaluationInput contains the artefacts needed to grade a coding submission.
type EvaluationInput struct {
	TaskTitle        string
	Prompt           string
	StarterCode      string
	Language         string
	SubmissionSource string
	SubmissionOutput string
	ExpectedOutput   string
	AdditionalNotes  string
}

// EvaluationResult is the structured feedback returned by the AI evaluator.
type EvaluationResult struct {
	Score    float64                `json:"score"`
	Feedback string                 `json:"feedback"`
	Verdict  string                 `json:"verdict"`
	Details  map[string]interface{} `json:"details,omitempty"`
	Raw      map[string]interface{} `json:"raw,omitempty"`
}

// Evaluator describes an AI model capable of grading code submissions.
type Evaluator interface {
	Evaluate(ctx context.Context, input EvaluationInput) (EvaluationResult, error)
}
