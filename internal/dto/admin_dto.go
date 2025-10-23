package dto

import (
	"strconv"
	"time"

	"gorm.io/datatypes"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// PaginationMeta captures pagination metadata for list responses.
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// AdminStudentListRequest defines filters for listing students.
type AdminStudentListRequest struct {
	Page     int
	PageSize int
	Search   string
	Class    string
	Status   string
	Sort     string
}

// AdminStudentResponse serializes student data for admin endpoints.
type AdminStudentResponse struct {
	ID        uint            `json:"id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	Class     string          `json:"class"`
	Status    string          `json:"status"`
	Flagged   bool            `json:"flagged"`
	Notes     string          `json:"notes"`
	Flags     map[string]bool `json:"flags"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
}

// AdminStudentListResponse wraps a paginated student response.
type AdminStudentListResponse struct {
	Items      []AdminStudentResponse `json:"items"`
	Pagination PaginationMeta         `json:"pagination"`
}

// AdminStudentUpdateRequest captures partial update payloads for students.
type AdminStudentUpdateRequest struct {
	Name    *string         `json:"name" validate:"omitempty,min=1"`
	Email   *string         `json:"email" validate:"omitempty,email"`
	Class   *string         `json:"class" validate:"omitempty,min=1"`
	Status  *string         `json:"status" validate:"omitempty,oneof=active inactive archived suspended"`
	Flagged *bool           `json:"flagged"`
	Notes   *string         `json:"notes" validate:"omitempty,max=2000"`
	Flags   map[string]bool `json:"flags" validate:"omitempty,dive,keys,required,endkeys"`
}

// NewAdminStudentResponse converts a student model into a DTO.
func NewAdminStudentResponse(student models.Student) AdminStudentResponse {
	var deletedAt *time.Time
	if student.DeletedAt.Valid {
		t := student.DeletedAt.Time
		deletedAt = &t
	}

	return AdminStudentResponse{
		ID:        student.ID,
		Name:      student.Name,
		Email:     student.Email,
		Class:     student.Class,
		Status:    student.Status,
		Flagged:   student.Flagged,
		Notes:     student.Notes,
		Flags:     boolMapFromJSON(student.Flags),
		CreatedAt: student.CreatedAt,
		UpdatedAt: student.UpdatedAt,
		DeletedAt: deletedAt,
	}
}

// AdminAssignmentCreateRequest captures metadata for creating assignments from the admin panel.
type AdminAssignmentCreateRequest struct {
	Title       string             `json:"title" validate:"required,min=3"`
	Description string             `json:"description" validate:"omitempty,min=5"`
	DueDate     string             `json:"due_date" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	MaxScore    float64            `json:"max_score" validate:"required,gt=0"`
	Rubric      map[string]float64 `json:"rubric" validate:"omitempty,dive,keys,required,endkeys,gt=0"`
	FileURL     string             `json:"file_url" validate:"omitempty,url"`
}

// AdminAssignmentUpdateRequest allows patching assignment metadata.
type AdminAssignmentUpdateRequest struct {
	Title       *string            `json:"title" validate:"omitempty,min=3"`
	Description *string            `json:"description" validate:"omitempty,min=5"`
	DueDate     *string            `json:"due_date" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	MaxScore    *float64           `json:"max_score" validate:"omitempty,gt=0"`
	Rubric      map[string]float64 `json:"rubric" validate:"omitempty,dive,keys,required,endkeys,gt=0"`
	FileURL     *string            `json:"file_url" validate:"omitempty,url"`
}

// AdminAssignmentResponse serializes assignment data for admin clients.
type AdminAssignmentResponse struct {
	ID          uint               `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	DueDate     time.Time          `json:"due_date"`
	FileURL     string             `json:"file_url"`
	MaxScore    float64            `json:"max_score"`
	Rubric      map[string]float64 `json:"rubric"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// NewAdminAssignmentResponse converts a model into a DTO for admin clients.
func NewAdminAssignmentResponse(model models.Assignment) AdminAssignmentResponse {
	return AdminAssignmentResponse{
		ID:          model.ID,
		Title:       model.Title,
		Description: model.Description,
		DueDate:     model.DueDate,
		FileURL:     model.FileURL,
		MaxScore:    model.MaxScore,
		Rubric:      floatMapFromJSON(model.Rubric),
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

// AdminGradeSubmissionRequest captures payloads for grading submissions.
type AdminGradeSubmissionRequest struct {
	Score    float64 `json:"score" validate:"required,gte=0"`
	Feedback string  `json:"feedback" validate:"omitempty,max=5000"`
}

// GradeDistributionResponse represents aggregated grade buckets.
type GradeDistributionResponse map[string]int64

// WeeklyEngagementPoint captures submissions per week.
type WeeklyEngagementPoint struct {
	WeekStart   time.Time `json:"week_start"`
	Submissions int64     `json:"submissions"`
}

// AdminAnalyticsResponse aggregates analytics metrics for administrators.
type AdminAnalyticsResponse struct {
	ActiveStudents    int64                     `json:"active_students"`
	OnTimeSubmissions int64                     `json:"on_time_submissions"`
	LateSubmissions   int64                     `json:"late_submissions"`
	GradeDistribution GradeDistributionResponse `json:"grade_distribution"`
	WeeklyEngagement  []WeeklyEngagementPoint   `json:"weekly_engagement"`
	GeneratedAt       time.Time                 `json:"generated_at"`
	CacheHit          bool                      `json:"cache_hit"`
}

// AdminActivityListRequest defines filters for retrieving activity logs.
type AdminActivityListRequest struct {
	Page       int
	PageSize   int
	ActorID    uint
	Action     string
	EntityType string
}

// AdminActivityCreateRequest captures manual activity log creation payloads.
type AdminActivityCreateRequest struct {
	Action     string                 `json:"action" validate:"required,min=3"`
	EntityType string                 `json:"entity_type" validate:"required,min=2"`
	EntityID   *uint                  `json:"entity_id"`
	Metadata   map[string]interface{} `json:"metadata" validate:"omitempty"`
}

// AdminActivityResponse serializes activity log entries.
type AdminActivityResponse struct {
	ID         uint                   `json:"id"`
	ActorID    uint                   `json:"actor_id"`
	ActorRole  string                 `json:"actor_role"`
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityID   *uint                  `json:"entity_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

// AdminActivityListResponse wraps paginated activity logs.
type AdminActivityListResponse struct {
	Items      []AdminActivityResponse `json:"items"`
	Pagination PaginationMeta          `json:"pagination"`
}

func boolMapFromJSON(data datatypes.JSONMap) map[string]bool {
	result := make(map[string]bool)
	if data == nil {
		return result
	}
	for key, raw := range data {
		switch value := raw.(type) {
		case bool:
			result[key] = value
		case string:
			if parsed, err := strconv.ParseBool(value); err == nil {
				result[key] = parsed
			}
		case float64:
			result[key] = value != 0
		case int:
			result[key] = value != 0
		default:
			// ignore unsupported types
		}
	}
	return result
}

func floatMapFromJSON(data datatypes.JSONMap) map[string]float64 {
	result := make(map[string]float64)
	if data == nil {
		return result
	}
	for key, raw := range data {
		switch value := raw.(type) {
		case float64:
			result[key] = value
		case int:
			result[key] = float64(value)
		case int64:
			result[key] = float64(value)
		case string:
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				result[key] = parsed
			}
		}
	}
	return result
}

func metadataFromJSON(data datatypes.JSONMap) map[string]interface{} {
	if data == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}(data)
}

// NewAdminActivityResponse converts a model into an activity DTO.
func NewAdminActivityResponse(entry models.ActivityLog) AdminActivityResponse {
	return AdminActivityResponse{
		ID:         entry.ID,
		ActorID:    entry.ActorID,
		ActorRole:  entry.ActorRole,
		Action:     entry.Action,
		EntityType: entry.EntityType,
		EntityID:   entry.EntityID,
		Metadata:   metadataFromJSON(entry.Metadata),
		CreatedAt:  entry.CreatedAt,
	}
}
