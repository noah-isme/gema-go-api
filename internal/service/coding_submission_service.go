package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
	"github.com/noah-isme/gema-go-api/pkg/ai"
	dockerexec "github.com/noah-isme/gema-go-api/pkg/docker"
)

// CodingSubmissionService exposes coding submission operations.
type CodingSubmissionService interface {
	Submit(ctx context.Context, studentID uint, payload dto.CodingSubmissionRequest) (dto.CodingSubmissionResponse, error)
	Get(ctx context.Context, id uint, viewerID uint, role string) (dto.CodingSubmissionResponse, error)
	Evaluate(ctx context.Context, id uint, evaluatorID uint, role string) (dto.CodingEvaluationResponse, error)
}

// ErrCodingSubmissionNotFound indicates the submission cannot be located.
var ErrCodingSubmissionNotFound = errors.New("coding submission not found")

// ErrCodingSubmissionForbidden indicates the caller is not allowed to access the submission.
var ErrCodingSubmissionForbidden = errors.New("forbidden")

// ErrUnsupportedLanguage indicates the requested language is not allowed.
var ErrUnsupportedLanguage = errors.New("unsupported language")

// ErrEvaluatorUnavailable indicates the AI evaluator is not configured.
var ErrEvaluatorUnavailable = errors.New("evaluator unavailable")

// CodingSubmissionConfig describes execution configuration knobs.
type CodingSubmissionConfig struct {
	ExecutionTimeout time.Duration
	MemoryLimitMB    int
	CPUShares        int
	WorkspaceRoot    string
}

type languageConfig struct {
	Image    string
	FileName string
	Command  []string
}

type codingSubmissionService struct {
	submissions repository.CodingSubmissionRepository
	tasks       repository.CodingTaskRepository
	executor    dockerexec.Executor
	evaluator   ai.Evaluator
	validator   *validator.Validate
	logger      zerolog.Logger
	config      CodingSubmissionConfig
	languages   map[string]languageConfig
}

// NewCodingSubmissionService constructs a new coding submission service.
func NewCodingSubmissionService(submissionRepo repository.CodingSubmissionRepository, taskRepo repository.CodingTaskRepository, executor dockerexec.Executor, evaluator ai.Evaluator, validate *validator.Validate, logger zerolog.Logger, cfg CodingSubmissionConfig) CodingSubmissionService {
	if cfg.WorkspaceRoot == "" {
		cfg.WorkspaceRoot = os.TempDir()
	}

	service := &codingSubmissionService{
		submissions: submissionRepo,
		tasks:       taskRepo,
		executor:    executor,
		evaluator:   evaluator,
		validator:   validate,
		logger:      logger.With().Str("component", "coding_submission_service").Logger(),
		config:      cfg,
		languages: map[string]languageConfig{
			"python": {
				Image:    "python:3.11-alpine",
				FileName: "main.py",
				Command:  []string{"python", "main.py"},
			},
			"javascript": {
				Image:    "node:20-alpine",
				FileName: "main.js",
				Command:  []string{"node", "main.js"},
			},
			"go": {
				Image:    "golang:1.22-alpine",
				FileName: "main.go",
				Command:  []string{"sh", "-c", "go run main.go"},
			},
		},
	}

	return service
}

func (s *codingSubmissionService) Submit(ctx context.Context, studentID uint, payload dto.CodingSubmissionRequest) (dto.CodingSubmissionResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.CodingSubmissionResponse{}, err
	}

	language := strings.ToLower(strings.TrimSpace(payload.Language))
	langCfg, ok := s.languages[language]
	if !ok {
		return dto.CodingSubmissionResponse{}, ErrUnsupportedLanguage
	}

	task, err := s.tasks.GetByID(ctx, payload.TaskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CodingSubmissionResponse{}, ErrCodingTaskNotFound
		}
		return dto.CodingSubmissionResponse{}, err
	}

	workspace, err := os.MkdirTemp(s.config.WorkspaceRoot, "submission-")
	if err != nil {
		return dto.CodingSubmissionResponse{}, fmt.Errorf("create workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	filePath := filepath.Join(workspace, langCfg.FileName)
	if err := os.WriteFile(filePath, []byte(payload.Source), 0600); err != nil {
		return dto.CodingSubmissionResponse{}, fmt.Errorf("write source: %w", err)
	}

	req := dockerexec.ExecutionRequest{
		Image:           langCfg.Image,
		Cmd:             langCfg.Command,
		Timeout:         s.config.ExecutionTimeout,
		Workspace:       workspace,
		WorkingDir:      "/workspace",
		MemoryLimitMB:   int64(s.config.MemoryLimitMB),
		CPUShares:       int64(s.config.CPUShares),
		NetworkDisabled: true,
		ReadOnlyFS:      false,
	}

	result, execErr := s.executor.Run(ctx, req)

	submission := models.CodingSubmission{
		TaskID:    payload.TaskID,
		StudentID: studentID,
		Language:  language,
		Source:    payload.Source,
		Output:    result.Stdout,
		Error:     combineErrors(result.Stderr, execErr),
		CPUTimeMs: result.Duration.Milliseconds(),
		MemoryKB:  result.MemoryUsageBytes / 1024,
	}

	switch {
	case execErr != nil && result.TimedOut:
		submission.Status = models.CodingSubmissionStatusTimeout
	case execErr != nil:
		submission.Status = models.CodingSubmissionStatusFailed
	case result.ExitCode != 0:
		submission.Status = models.CodingSubmissionStatusFailed
		if submission.Error == "" {
			submission.Error = fmt.Sprintf("process exited with code %d", result.ExitCode)
		}
	default:
		submission.Status = models.CodingSubmissionStatusCompleted
	}

	if err := s.submissions.Create(ctx, &submission); err != nil {
		return dto.CodingSubmissionResponse{}, err
	}

	submission.Task = task
	response := dto.NewCodingSubmissionResponse(submission, true)
	return response, nil
}

func (s *codingSubmissionService) Get(ctx context.Context, id uint, viewerID uint, role string) (dto.CodingSubmissionResponse, error) {
	submission, err := s.submissions.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CodingSubmissionResponse{}, ErrCodingSubmissionNotFound
		}
		return dto.CodingSubmissionResponse{}, err
	}

	includeSource := s.canViewSource(viewerID, role, submission)
	if !includeSource {
		submission.Source = ""
	}

	return dto.NewCodingSubmissionResponse(submission, includeSource), nil
}

func (s *codingSubmissionService) Evaluate(ctx context.Context, id uint, evaluatorID uint, role string) (dto.CodingEvaluationResponse, error) {
	if !s.canEvaluate(role) {
		return dto.CodingEvaluationResponse{}, ErrCodingSubmissionForbidden
	}
	if s.evaluator == nil {
		return dto.CodingEvaluationResponse{}, ErrEvaluatorUnavailable
	}

	submission, err := s.submissions.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CodingEvaluationResponse{}, ErrCodingSubmissionNotFound
		}
		return dto.CodingEvaluationResponse{}, err
	}

	task := submission.Task

	result, err := s.evaluator.Evaluate(ctx, ai.EvaluationInput{
		TaskTitle:        task.Title,
		Prompt:           task.Prompt,
		StarterCode:      task.StarterCode,
		Language:         submission.Language,
		SubmissionSource: submission.Source,
		SubmissionOutput: submission.Output,
		ExpectedOutput:   task.ExpectedOutput,
	})
	if err != nil {
		return dto.CodingEvaluationResponse{}, err
	}

	evaluation := models.CodingEvaluation{
		SubmissionID: submission.ID,
		Score:        result.Score,
		Verdict:      result.Verdict,
		Feedback:     result.Feedback,
		Provider:     s.providerName(),
		Details:      datatypes.JSONMap(result.Details),
		Raw:          datatypes.JSONMap(result.Raw),
	}

	if err := s.submissions.SaveEvaluation(ctx, &evaluation); err != nil {
		return dto.CodingEvaluationResponse{}, err
	}

	submission.Status = models.CodingSubmissionStatusEvaluated
	if err := s.submissions.Update(ctx, &submission); err != nil {
		s.logger.Error().Err(err).Uint("submission_id", submission.ID).Msg("failed to update submission status")
	}

	return dto.NewCodingEvaluationResponse(evaluation), nil
}

func (s *codingSubmissionService) canViewSource(viewerID uint, role string, submission models.CodingSubmission) bool {
	if viewerID != 0 && viewerID == submission.StudentID {
		return true
	}
	role = strings.ToLower(role)
	return role == "teacher" || role == "admin"
}

func (s *codingSubmissionService) canEvaluate(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return role == "teacher" || role == "admin"
}

func (s *codingSubmissionService) providerName() string {
	switch s := s.evaluator.(type) {
	case *ai.OpenAIEvaluator:
		return "openai"
	case *ai.AnthropicEvaluator:
		return "anthropic"
	default:
		_ = s
		return "unknown"
	}
}

func combineErrors(stderr string, execErr error) string {
	if execErr == nil {
		return strings.TrimSpace(stderr)
	}
	if stderr == "" {
		return execErr.Error()
	}
	return strings.TrimSpace(fmt.Sprintf("%s\n%s", stderr, execErr.Error()))
}
