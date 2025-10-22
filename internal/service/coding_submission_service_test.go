package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
	"github.com/noah-isme/gema-go-api/pkg/ai"
	dockerexec "github.com/noah-isme/gema-go-api/pkg/docker"
)

type stubSubmissionRepo struct {
	created    *models.CodingSubmission
	updated    *models.CodingSubmission
	evaluation *models.CodingEvaluation
	stored     models.CodingSubmission
	err        error
}

func (s *stubSubmissionRepo) Create(ctx context.Context, submission *models.CodingSubmission) error {
	if s.err != nil {
		return s.err
	}
	if submission.ID == 0 {
		submission.ID = 1
	}
	clone := *submission
	s.created = &clone
	s.stored = clone
	return nil
}

func (s *stubSubmissionRepo) Update(ctx context.Context, submission *models.CodingSubmission) error {
	if s.err != nil {
		return s.err
	}
	clone := *submission
	s.updated = &clone
	s.stored = clone
	return nil
}

func (s *stubSubmissionRepo) GetByID(ctx context.Context, id uint) (models.CodingSubmission, error) {
	if s.err != nil {
		return models.CodingSubmission{}, s.err
	}
	if s.stored.ID == 0 {
		return models.CodingSubmission{}, gorm.ErrRecordNotFound
	}
	return s.stored, nil
}

func (s *stubSubmissionRepo) SaveEvaluation(ctx context.Context, evaluation *models.CodingEvaluation) error {
	if s.err != nil {
		return s.err
	}
	clone := *evaluation
	s.evaluation = &clone
	return nil
}

type stubTaskRepo struct {
	task models.CodingTask
	err  error
}

func (s *stubTaskRepo) List(ctx context.Context, query repository.CodingTaskQuery) ([]models.CodingTask, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *stubTaskRepo) GetByID(ctx context.Context, id uint) (models.CodingTask, error) {
	if s.err != nil {
		return models.CodingTask{}, s.err
	}
	if s.task.ID == 0 {
		return models.CodingTask{}, gorm.ErrRecordNotFound
	}
	return s.task, nil
}

type stubExecutor struct {
	result dockerexec.ExecutionResult
	err    error
}

func (s stubExecutor) Run(ctx context.Context, req dockerexec.ExecutionRequest) (dockerexec.ExecutionResult, error) {
	return s.result, s.err
}

type stubEvaluator struct {
	result ai.EvaluationResult
	err    error
}

func (s stubEvaluator) Evaluate(ctx context.Context, input ai.EvaluationInput) (ai.EvaluationResult, error) {
	if s.err != nil {
		return ai.EvaluationResult{}, s.err
	}
	return s.result, nil
}

func TestCodingSubmissionServiceRejectsUnsupportedLanguage(t *testing.T) {
	svc := NewCodingSubmissionService(&stubSubmissionRepo{}, &stubTaskRepo{}, stubExecutor{}, nil, validator.New(validator.WithRequiredStructEnabled()), zerolog.Nop(), CodingSubmissionConfig{})

	_, err := svc.Submit(context.Background(), 1, dto.CodingSubmissionRequest{TaskID: 1, Language: "ruby", Source: "puts 'hi'"})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnsupportedLanguage))
}

func TestCodingSubmissionServiceHandlesTimeout(t *testing.T) {
	repo := &stubSubmissionRepo{}
	taskRepo := &stubTaskRepo{task: models.CodingTask{ID: 1, Title: "FizzBuzz"}}
	exec := stubExecutor{result: dockerexec.ExecutionResult{Stdout: "", Stderr: "", Duration: time.Second, TimedOut: true}, err: fmt.Errorf("timeout")}
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewCodingSubmissionService(repo, taskRepo, exec, nil, validate, zerolog.Nop(), CodingSubmissionConfig{ExecutionTimeout: time.Second})

	resp, err := svc.Submit(context.Background(), 10, dto.CodingSubmissionRequest{TaskID: 1, Language: "python", Source: "print('hi')"})
	require.NoError(t, err)
	require.Equal(t, models.CodingSubmissionStatusTimeout, repo.created.Status)
	require.Equal(t, repo.created.ID, resp.ID)
}

func TestCodingSubmissionServiceEvaluateStoresResult(t *testing.T) {
	submissionRepo := &stubSubmissionRepo{stored: models.CodingSubmission{ID: 5, TaskID: 1, StudentID: 2, Language: "python", Source: "print('hi')", Task: models.CodingTask{ID: 1, Title: "Fizz", Prompt: "prompt"}}}
	taskRepo := &stubTaskRepo{task: models.CodingTask{ID: 1, Title: "Fizz", Prompt: "prompt"}}
	evaluator := stubEvaluator{result: ai.EvaluationResult{Score: 0.9, Feedback: "Great", Verdict: "pass", Details: map[string]interface{}{"correctness": 1}}}
	svc := NewCodingSubmissionService(submissionRepo, taskRepo, stubExecutor{}, evaluator, validator.New(validator.WithRequiredStructEnabled()), zerolog.Nop(), CodingSubmissionConfig{})

	eval, err := svc.Evaluate(context.Background(), 5, 1, "teacher")
	require.NoError(t, err)
	require.NotNil(t, submissionRepo.evaluation)
	require.InDelta(t, 0.9, submissionRepo.evaluation.Score, 0.001)
	require.Equal(t, datatypes.JSONMap{"correctness": 1}, submissionRepo.evaluation.Details)
	require.Equal(t, "pass", eval.Verdict)
}

func TestCodingSubmissionServiceEvaluateRequiresEvaluator(t *testing.T) {
	submissionRepo := &stubSubmissionRepo{stored: models.CodingSubmission{ID: 5, TaskID: 1, StudentID: 2, Language: "python", Source: "print('hi')", Task: models.CodingTask{ID: 1, Title: "Fizz"}}}
	taskRepo := &stubTaskRepo{task: models.CodingTask{ID: 1, Title: "Fizz"}}
	svc := NewCodingSubmissionService(submissionRepo, taskRepo, stubExecutor{}, nil, validator.New(validator.WithRequiredStructEnabled()), zerolog.Nop(), CodingSubmissionConfig{})

	_, err := svc.Evaluate(context.Background(), 5, 1, "teacher")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrEvaluatorUnavailable))
}
