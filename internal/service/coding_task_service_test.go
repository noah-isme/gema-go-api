package service

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

type stubCodingTaskRepo struct {
	tasks []models.CodingTask
	err   error
	last  repository.CodingTaskQuery
}

func (s *stubCodingTaskRepo) List(ctx context.Context, query repository.CodingTaskQuery) ([]models.CodingTask, int64, error) {
	s.last = query
	if s.err != nil {
		return nil, 0, s.err
	}
	return s.tasks, int64(len(s.tasks)), nil
}

func (s *stubCodingTaskRepo) GetByID(ctx context.Context, id uint) (models.CodingTask, error) {
	if s.err != nil {
		return models.CodingTask{}, s.err
	}
	for _, task := range s.tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return models.CodingTask{}, gorm.ErrRecordNotFound
}

func TestCodingTaskServiceListAppliesDefaults(t *testing.T) {
	repo := &stubCodingTaskRepo{tasks: []models.CodingTask{{ID: 1, Title: "FizzBuzz", Prompt: "  prompt  "}}}
	svc := NewCodingTaskService(repo, zerolog.Nop())

	resp, err := svc.List(context.Background(), dto.CodingTaskFilter{})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, 1, resp.Pagination.Page)
	require.Equal(t, 20, resp.Pagination.PageSize)
	require.Equal(t, "prompt", resp.Items[0].Prompt)
}

func TestCodingTaskServiceGetNotFound(t *testing.T) {
	repo := &stubCodingTaskRepo{}
	svc := NewCodingTaskService(repo, zerolog.Nop())

	_, err := svc.Get(context.Background(), 10)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrCodingTaskNotFound))
}
