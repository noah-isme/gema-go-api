package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	aiDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "gema",
		Subsystem: "ai",
		Name:      "evaluation_duration_seconds",
		Help:      "Duration of AI evaluation requests",
	}, []string{"model"})

	aiFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gema",
		Subsystem: "ai",
		Name:      "evaluation_failures_total",
		Help:      "Number of AI evaluation failures",
	}, []string{"model"})
)

// OpenAIConfig defines configuration options for the OpenAI evaluator.
type OpenAIConfig struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float32
	Logger      zerolog.Logger
}

// OpenAIEvaluator implements Evaluator against the OpenAI chat completion API.
type OpenAIEvaluator struct {
	client *openai.Client
	cfg    OpenAIConfig
	tracer trace.Tracer
	logger zerolog.Logger
}

// NewOpenAIEvaluator builds a new evaluator using the provided configuration.
func NewOpenAIEvaluator(cfg OpenAIConfig) (*OpenAIEvaluator, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai api key is required")
	}

	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}

	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 512
	}

	tracer := otel.Tracer("github.com/noah-isme/gema-go-api/pkg/ai/openai")
	logger := cfg.Logger
	if logger.GetLevel() == zerolog.Disabled {
		logger = zerolog.Nop()
	}

	config := openai.DefaultConfig(cfg.APIKey)
	client := openai.NewClientWithConfig(config)

	return &OpenAIEvaluator{
		client: client,
		cfg:    cfg,
		tracer: tracer,
		logger: logger,
	}, nil
}

// Evaluate sends the evaluation request to OpenAI and parses the response.
func (e *OpenAIEvaluator) Evaluate(parent context.Context, input EvaluationInput) (EvaluationResult, error) {
	ctx, span := e.tracer.Start(parent, "openai.evaluate", trace.WithAttributes(
		attribute.String("model", e.cfg.Model),
	))
	defer span.End()

	start := time.Now()
	request := openai.ChatCompletionRequest{
		Model:       e.cfg.Model,
		MaxTokens:   e.cfg.MaxTokens,
		Temperature: e.cfg.Temperature,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: evaluatorSystemPrompt(),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: buildUserPrompt(input),
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
	}

	resp, err := e.client.CreateChatCompletion(ctx, request)
	duration := time.Since(start)
	aiDuration.WithLabelValues(e.cfg.Model).Observe(duration.Seconds())
	if err != nil {
		aiFailures.WithLabelValues(e.cfg.Model).Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return EvaluationResult{}, fmt.Errorf("openai evaluate: %w", err)
	}

	if len(resp.Choices) == 0 {
		err := fmt.Errorf("no choices returned from openai")
		aiFailures.WithLabelValues(e.cfg.Model).Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return EvaluationResult{}, err
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	result, err := parseEvaluationResponse(content)
	if err != nil {
		aiFailures.WithLabelValues(e.cfg.Model).Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return EvaluationResult{}, err
	}

	result.Raw = map[string]interface{}{
		"usage": resp.Usage,
	}

	return result, nil
}

func evaluatorSystemPrompt() string {
	return "You are an automated code reviewer. Respond with a JSON object containing score (0-1), verdict, feedback, and opti" +
		"onal details object breaking down the score. Focus on correctness, code quality, and edge cases."
}

func buildUserPrompt(input EvaluationInput) string {
	builder := strings.Builder{}
	builder.WriteString("# Task\n")
	builder.WriteString(input.TaskTitle)
	builder.WriteString("\n\n## Prompt\n")
	builder.WriteString(input.Prompt)
	builder.WriteString("\n\n## Starter Code\n")
	builder.WriteString(input.StarterCode)
	builder.WriteString("\n\n## Language\n")
	builder.WriteString(input.Language)
	builder.WriteString("\n\n## Submission\n")
	builder.WriteString(input.SubmissionSource)
	builder.WriteString("\n\n## Program Output\n")
	builder.WriteString(input.SubmissionOutput)
	builder.WriteString("\n\n## Expected Behaviour\n")
	builder.WriteString(input.ExpectedOutput)
	if input.AdditionalNotes != "" {
		builder.WriteString("\n\n## Notes\n")
		builder.WriteString(input.AdditionalNotes)
	}
	builder.WriteString("\nReturn JSON.")
	return builder.String()
}

func parseEvaluationResponse(content string) (EvaluationResult, error) {
	type payload struct {
		Score    float64                `json:"score"`
		Feedback string                 `json:"feedback"`
		Verdict  string                 `json:"verdict"`
		Details  map[string]interface{} `json:"details"`
	}

	var data payload
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return EvaluationResult{}, fmt.Errorf("parse evaluation json: %w", err)
	}

	if data.Score < 0 {
		data.Score = 0
	}
	if data.Score > 1 {
		data.Score = 1
	}

	return EvaluationResult{
		Score:    data.Score,
		Feedback: data.Feedback,
		Verdict:  data.Verdict,
		Details:  data.Details,
	}, nil
}
