package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// StudentDashboardService produces aggregated dashboard metrics.
type StudentDashboardService interface {
	GetDashboard(ctx context.Context, studentID uint) (dto.StudentDashboardResponse, error)
}

type studentDashboardService struct {
	assignments repository.AssignmentRepository
	submissions repository.SubmissionRepository
	cache       *redis.Client
	cacheTTL    time.Duration
	logger      zerolog.Logger
	now         func() time.Time
}

// NewStudentDashboardService builds the dashboard aggregator.
func NewStudentDashboardService(assignments repository.AssignmentRepository, submissions repository.SubmissionRepository, cache *redis.Client, ttl time.Duration, logger zerolog.Logger) StudentDashboardService {
	return &studentDashboardService{
		assignments: assignments,
		submissions: submissions,
		cache:       cache,
		cacheTTL:    ttl,
		logger:      logger.With().Str("component", "student_dashboard_service").Logger(),
		now:         time.Now,
	}
}

func (s *studentDashboardService) GetDashboard(ctx context.Context, studentID uint) (dto.StudentDashboardResponse, error) {
	cacheKey := fmt.Sprintf("dashboard:student:%d", studentID)

	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey).Result(); err == nil {
			var response dto.StudentDashboardResponse
			if unmarshalErr := json.Unmarshal([]byte(cached), &response); unmarshalErr == nil {
				s.logger.Debug().Uint("student_id", studentID).Msg("dashboard cache hit")
				return response, nil
			}
		} else if err != redis.Nil {
			s.logger.Warn().Err(err).Msg("failed to read dashboard cache")
		}
	}

	assignments, err := s.assignments.List(ctx)
	if err != nil {
		return dto.StudentDashboardResponse{}, err
	}

	filter := repository.SubmissionFilter{StudentID: &studentID}
	submissions, err := s.submissions.List(ctx, filter)
	if err != nil {
		return dto.StudentDashboardResponse{}, err
	}

	response := s.buildResponse(assignments, submissions)

	if s.cache != nil {
		payload, err := json.Marshal(response)
		if err == nil {
			if err := s.cache.Set(ctx, cacheKey, payload, s.cacheTTL).Err(); err != nil {
				s.logger.Warn().Err(err).Msg("failed to store dashboard cache")
			}
		}
	}

	return response, nil
}

func (s *studentDashboardService) buildResponse(assignments []models.Assignment, submissions []models.Submission) dto.StudentDashboardResponse {
	now := s.now()
	submissionByAssignment := map[uint]models.Submission{}
	for _, submission := range submissions {
		if _, exists := submissionByAssignment[submission.AssignmentID]; !exists {
			submissionByAssignment[submission.AssignmentID] = submission
		}
	}

	summary := dto.ProgressSummary{}
	progress := make([]dto.AssignmentProgress, 0, len(assignments))
	var gradeTotal float64
	var gradedCount int

	for _, assignment := range assignments {
		summary.TotalAssignments++
		submission, submitted := submissionByAssignment[assignment.ID]
		assignmentOverdue := assignment.IsPastDue(now)

		status := "pending"
		var submissionID *uint
		submissionURL := ""
		var grade *float64
		feedback := ""
		updatedAt := assignment.UpdatedAt

		if submitted {
			submissionID = &submission.ID
			submissionURL = submission.FileURL
			feedback = submission.Feedback
			updatedAt = submission.UpdatedAt
			summary.Submitted++

			switch submission.Status {
			case models.SubmissionStatusGraded:
				status = models.SubmissionStatusGraded
				summary.Graded++
				if submission.Grade != nil {
					gradeTotal += *submission.Grade
					gradedCount++
					grade = submission.Grade
				}
			default:
				status = models.SubmissionStatusSubmitted
				summary.Pending++
			}
		} else {
			summary.Pending++
			if assignmentOverdue {
				summary.Overdue++
			}
		}

		if submitted && assignmentOverdue && submission.Status != models.SubmissionStatusGraded {
			summary.Overdue++
		}

		progress = append(progress, dto.AssignmentProgress{
			AssignmentID:  assignment.ID,
			Title:         assignment.Title,
			DueDate:       assignment.DueDate,
			FileURL:       assignment.FileURL,
			Status:        status,
			SubmissionID:  submissionID,
			SubmissionURL: submissionURL,
			Grade:         grade,
			Feedback:      feedback,
			UpdatedAt:     updatedAt,
			Overdue:       assignmentOverdue && (status != models.SubmissionStatusGraded),
		})
	}

	if gradedCount > 0 {
		summary.AverageGrade = gradeTotal / float64(gradedCount)
	}

	if summary.TotalAssignments > 0 {
		summary.CompletionRate = (float64(summary.Graded) / float64(summary.TotalAssignments)) * 100
	}

	pendingAssignments := make([]dto.AssignmentProgress, 0)
	for _, item := range progress {
		if item.Status != models.SubmissionStatusGraded {
			pendingAssignments = append(pendingAssignments, item)
		}
	}

	activities := make([]dto.SubmissionActivity, 0, min(5, len(submissions)))
	for idx, submission := range submissions {
		if idx >= 5 {
			break
		}
		activities = append(activities, dto.SubmissionActivity{
			SubmissionID:   submission.ID,
			AssignmentID:   submission.AssignmentID,
			AssignmentName: submission.Assignment.Title,
			Status:         submission.Status,
			Grade:          submission.Grade,
			Feedback:       submission.Feedback,
			CreatedAt:      submission.CreatedAt,
			UpdatedAt:      submission.UpdatedAt,
		})
	}

	return dto.StudentDashboardResponse{
		Summary:           summary,
		Pending:           pendingAssignments,
		RecentSubmissions: activities,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
