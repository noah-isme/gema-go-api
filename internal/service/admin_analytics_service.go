package service

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

// AdminAnalyticsService aggregates analytics for the admin dashboard.
type AdminAnalyticsService interface {
	GetSummary(ctx context.Context) (dto.AdminAnalyticsResponse, error)
}

type adminAnalyticsService struct {
	repo     repository.AdminAnalyticsRepository
	cache    *redis.Client
	cacheTTL time.Duration
	logger   zerolog.Logger
	now      func() time.Time
}

// NewAdminAnalyticsService constructs the analytics service.
func NewAdminAnalyticsService(repo repository.AdminAnalyticsRepository, cache *redis.Client, ttl time.Duration, logger zerolog.Logger) AdminAnalyticsService {
	return &adminAnalyticsService{
		repo:     repo,
		cache:    cache,
		cacheTTL: ttl,
		logger:   logger.With().Str("component", "admin_analytics_service").Logger(),
		now:      time.Now,
	}
}

func (s *adminAnalyticsService) GetSummary(ctx context.Context) (dto.AdminAnalyticsResponse, error) {
	const cacheKey = "analytics:summary"

	if s.cache != nil {
		cached, err := s.cache.Get(ctx, cacheKey).Result()
		if err == nil {
			var response dto.AdminAnalyticsResponse
			if unmarshalErr := json.Unmarshal([]byte(cached), &response); unmarshalErr == nil {
				response.CacheHit = true
				return response, nil
			}
		} else if err != redis.Nil {
			s.logger.Warn().Err(err).Msg("failed to read analytics cache")
		}
	}

	activeCount, err := s.repo.CountActiveStudents(ctx)
	if err != nil {
		return dto.AdminAnalyticsResponse{}, err
	}

	submissions, err := s.repo.ListSubmissionsWithAssignments(ctx)
	if err != nil {
		return dto.AdminAnalyticsResponse{}, err
	}

	summary := s.buildSummary(activeCount, submissions)

	if s.cache != nil {
		payload, err := json.Marshal(summary)
		if err == nil {
			if err := s.cache.Set(ctx, cacheKey, payload, s.cacheTTL).Err(); err != nil {
				s.logger.Warn().Err(err).Msg("failed to store analytics cache")
			}
		}
	}

	return summary, nil
}

func (s *adminAnalyticsService) buildSummary(activeCount int64, submissions []models.Submission) dto.AdminAnalyticsResponse {
	now := s.now()
	onTime := int64(0)
	late := int64(0)
	distribution := dto.GradeDistributionResponse{
		"90-100": 0,
		"75-89":  0,
		"60-74":  0,
		"0-59":   0,
	}

	weekly := map[time.Time]int64{}
	cutoff := now.AddDate(0, 0, -56)

	for _, submission := range submissions {
		dueDate := submission.Assignment.DueDate
		if !dueDate.IsZero() && !submission.CreatedAt.After(dueDate) {
			onTime++
		} else if !dueDate.IsZero() {
			late++
		}

		maxScore := submission.Assignment.MaxScore
		if maxScore <= 0 {
			maxScore = 100
		}
		if submission.Grade != nil {
			percent := (*submission.Grade / maxScore) * 100
			switch {
			case percent >= 90:
				distribution["90-100"]++
			case percent >= 75:
				distribution["75-89"]++
			case percent >= 60:
				distribution["60-74"]++
			default:
				distribution["0-59"]++
			}
		}

		if submission.CreatedAt.After(cutoff) {
			week := startOfWeek(submission.CreatedAt)
			weekly[week]++
		}
	}

	weeks := make([]time.Time, 0, len(weekly))
	for week := range weekly {
		weeks = append(weeks, week)
	}
	sort.Slice(weeks, func(i, j int) bool { return weeks[i].Before(weeks[j]) })

	engagement := make([]dto.WeeklyEngagementPoint, 0, len(weeks))
	for _, week := range weeks {
		engagement = append(engagement, dto.WeeklyEngagementPoint{WeekStart: week, Submissions: weekly[week]})
	}

	return dto.AdminAnalyticsResponse{
		ActiveStudents:    activeCount,
		OnTimeSubmissions: onTime,
		LateSubmissions:   late,
		GradeDistribution: distribution,
		WeeklyEngagement:  engagement,
		GeneratedAt:       now,
		CacheHit:          false,
	}
}

func startOfWeek(t time.Time) time.Time {
	utc := t.UTC()
	weekday := int(utc.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := utc.AddDate(0, 0, -(weekday - 1))
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
}
