package ai

import (
	"context"
	"fmt"
)

// AnthropicConfig placeholder for anthropic integration configuration.
type AnthropicConfig struct {
	APIKey string
	Model  string
}

// AnthropicEvaluator is a stub implementation that can be expanded once the SDK is available.
type AnthropicEvaluator struct{}

// NewAnthropicEvaluator constructs a new stub evaluator.
func NewAnthropicEvaluator(cfg AnthropicConfig) (*AnthropicEvaluator, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic api key is required")
	}
	return &AnthropicEvaluator{}, nil
}

// Evaluate is not yet implemented for Anthropic models.
func (a *AnthropicEvaluator) Evaluate(ctx context.Context, input EvaluationInput) (EvaluationResult, error) {
	return EvaluationResult{}, fmt.Errorf("anthropic evaluator not implemented")
}
