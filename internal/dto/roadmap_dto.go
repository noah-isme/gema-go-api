package dto

import "time"

// RoadmapStageResponse serializes roadmap stage payloads.
type RoadmapStageResponse struct {
	ID             uint              `json:"id"`
	Slug           string            `json:"slug"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Sequence       int               `json:"sequence"`
	EstimatedHours int               `json:"estimated_hours"`
	Icon           string            `json:"icon"`
	Tags           []string          `json:"tags"`
	Skills         map[string]string `json:"skills"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// RoadmapStageListResult wraps paginated roadmap stages.
type RoadmapStageListResult struct {
	Items      []RoadmapStageResponse `json:"items"`
	Pagination PaginationMeta         `json:"pagination"`
	Filters    RoadmapStageFilters    `json:"filters"`
	CacheHit   bool                   `json:"cache_hit"`
}

// RoadmapStageFilters describes applied filters.
type RoadmapStageFilters struct {
	Search string   `json:"search,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Sort   string   `json:"sort,omitempty"`
}

// RoadmapStageListRequest captures query params.
type RoadmapStageListRequest struct {
	Page     int
	PageSize int
	Sort     string
	Search   string
	Tags     []string
}
