package service

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/models"
)

type fakeAnalyticsRepo struct {
	activeCount int64
	submissions []models.Submission
}

func (f *fakeAnalyticsRepo) CountActiveStudents(ctx context.Context) (int64, error) {
	return f.activeCount, nil
}

func (f *fakeAnalyticsRepo) ListSubmissionsWithAssignments(ctx context.Context) ([]models.Submission, error) {
	return append([]models.Submission(nil), f.submissions...), nil
}

func (f *fakeAnalyticsRepo) ListSubmissionsSince(ctx context.Context, since time.Time) ([]models.Submission, error) {
	result := make([]models.Submission, 0)
	for _, submission := range f.submissions {
		if submission.CreatedAt.After(since) || submission.CreatedAt.Equal(since) {
			result = append(result, submission)
		}
	}
	return result, nil
}

func TestAdminAnalyticsServiceCaching(t *testing.T) {
	server, err := miniredis.Run()
	require.NoError(t, err)
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	now := time.Now()
	grade := 85.0
	repo := &fakeAnalyticsRepo{
		activeCount: 5,
		submissions: []models.Submission{
			{
				ID:           1,
				AssignmentID: 1,
				StudentID:    1,
				CreatedAt:    now.Add(-24 * time.Hour),
				Grade:        &grade,
				Assignment: models.Assignment{
					ID:       1,
					MaxScore: 100,
					DueDate:  now.Add(-12 * time.Hour),
				},
			},
			{
				ID:           2,
				AssignmentID: 2,
				StudentID:    2,
				CreatedAt:    now.Add(-6 * time.Hour),
				Assignment: models.Assignment{
					ID:       2,
					MaxScore: 100,
					DueDate:  now.Add(-12 * time.Hour),
				},
			},
		},
	}

	svc := NewAdminAnalyticsService(repo, client, time.Minute, testLogger())

	summary, err := svc.GetSummary(context.Background())
	require.NoError(t, err)
	require.False(t, summary.CacheHit)
	require.Equal(t, int64(5), summary.ActiveStudents)
	require.Equal(t, int64(1), summary.OnTimeSubmissions)
	require.Equal(t, int64(1), summary.LateSubmissions)

	repo.activeCount = 10
	summaryCached, err := svc.GetSummary(context.Background())
	require.NoError(t, err)
	require.True(t, summaryCached.CacheHit)
	require.Equal(t, summary.ActiveStudents, summaryCached.ActiveStudents)
}
