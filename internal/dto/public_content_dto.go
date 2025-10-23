package dto

import "time"

// ActivityFeedRequest describes the incoming query for active activities.
type ActivityFeedRequest struct {
	Page     int
	PageSize int
	UserID   *uint
	Type     string
	Action   string
}

// ActivityFeedItem represents a single activity entry for public consumption.
type ActivityFeedItem struct {
	ID         uint                   `json:"id"`
	ActorID    uint                   `json:"actor_id"`
	ActorRole  string                 `json:"actor_role"`
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityID   *uint                  `json:"entity_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ActivityFeedResponse wraps paginated activity items.
type ActivityFeedResponse struct {
	Items      []ActivityFeedItem `json:"items"`
	Pagination PaginationMeta     `json:"pagination"`
	CacheHit   bool               `json:"cache_hit"`
}

// AnnouncementResponse represents an announcement payload returned to the frontend.
type AnnouncementResponse struct {
	ID        uint       `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	StartsAt  time.Time  `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
	IsPinned  bool       `json:"is_pinned"`
	CreatedAt time.Time  `json:"created_at"`
}

// AnnouncementListResponse contains paginated announcements.
type AnnouncementListResponse struct {
	Items      []AnnouncementResponse `json:"items"`
	Pagination PaginationMeta         `json:"pagination"`
	CacheHit   bool                   `json:"cache_hit"`
}

// GalleryItemResponse represents an item in the gallery feed.
type GalleryItemResponse struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Caption   string    `json:"caption"`
	ImageURL  string    `json:"image_url"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

// GalleryListResponse contains paginated gallery items.
type GalleryListResponse struct {
	Items      []GalleryItemResponse `json:"items"`
	Pagination PaginationMeta        `json:"pagination"`
}

// ContactRequest defines the expected payload for the contact form endpoint.
type ContactRequest struct {
	Name      string `json:"name" validate:"required,min=2,max=120"`
	Email     string `json:"email" validate:"required,email,max=160"`
	Message   string `json:"message" validate:"required,min=10,max=2000"`
	Source    string `json:"source" validate:"omitempty,max=60"`
	Honeypot  string `json:"_note"`
	Checksum  string `json:"-"`
	IPAddress string `json:"-"`
	UserID    *uint  `json:"-"`
}

// ContactResponse communicates the status of the submission processing.
type ContactResponse struct {
	ReferenceID string `json:"reference_id"`
	Status      string `json:"status"`
}

// UploadResponse describes the stored asset metadata returned to the client.
type UploadResponse struct {
	URL       string `json:"url"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
	Checksum  string `json:"checksum"`
	FileName  string `json:"file_name"`
}

// SeedRequest contains optional overrides for seed operations.
type SeedRequest struct {
	Force bool `json:"force"`
}
