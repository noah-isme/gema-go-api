package observability

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	registerOnce                sync.Once
	adminRequestsTotal          *prometheus.CounterVec
	adminLatencySeconds         *prometheus.HistogramVec
	adminErrorsTotal            *prometheus.CounterVec
	chatConnectionsTotal        prometheus.Counter
	chatDisconnectsTotal        prometheus.Counter
	chatMessagesSent            *prometheus.CounterVec
	sseClientsActive            prometheus.Gauge
	notificationsPublishedTotal *prometheus.CounterVec
	realtimeErrorsTotal         *prometheus.CounterVec
	activeActivitiesRequests    *prometheus.CounterVec
	activeActivitiesLatency     prometheus.Histogram
	announcementsRequests       *prometheus.CounterVec
	announcementsLatency        prometheus.Histogram
	galleryRequests             *prometheus.CounterVec
	galleryLatency              prometheus.Histogram
	roadmapRequests             *prometheus.CounterVec
	roadmapLatency              prometheus.Histogram
	dashboardRequests           *prometheus.CounterVec
	dashboardLatency            prometheus.Histogram
	contactSubmissions          *prometheus.CounterVec
	uploadRequests              *prometheus.CounterVec
	uploadRejected              *prometheus.CounterVec
	uploadLatency               prometheus.Histogram
)

// RegisterMetrics initialises the Prometheus collectors used for admin observability.
func RegisterMetrics() {
	registerOnce.Do(func() {
		adminRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "admin_requests_total",
			Help: "Total number of admin API requests served.",
		}, []string{"method", "route", "status"})

		adminLatencySeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "admin_latency_seconds",
			Help:    "Latency distribution for admin API requests.",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0},
		}, []string{"method", "route"})

		adminErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "admin_errors_total",
			Help: "Total number of error responses returned by admin endpoints.",
		}, []string{"method", "route", "status"})

		chatConnectionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "chat_connections_total",
			Help: "Total number of websocket chat connections established.",
		})

		chatDisconnectsTotal = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "chat_disconnects_total",
			Help: "Total number of websocket chat disconnects observed.",
		})

		chatMessagesSent = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "chat_messages_sent",
			Help: "Total chat messages sent segmented by type.",
		}, []string{"type"})

		sseClientsActive = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sse_clients_active",
			Help: "Number of active SSE clients streaming notifications.",
		})

		notificationsPublishedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notifications_published_total",
			Help: "Total number of notifications published segmented by type.",
		}, []string{"type"})

		realtimeErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "realtime_errors_total",
			Help: "Total number of realtime streaming errors segmented by component and reason.",
		}, []string{"component", "reason"})

		activeActivitiesRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "active_activities_requests_total",
			Help: "Total number of active activity feed requests.",
		}, []string{"result"})

		activeActivitiesLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "active_activities_latency_seconds",
			Help:    "Latency distribution for active activities endpoint.",
			Buckets: prometheus.DefBuckets,
		})

		announcementsRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "announcements_requests_total",
			Help: "Total number of announcements requests served.",
		}, []string{"result"})

		announcementsLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "announcements_latency_seconds",
			Help:    "Latency distribution for announcements endpoint.",
			Buckets: prometheus.DefBuckets,
		})

		galleryRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gallery_requests_total",
			Help: "Total number of gallery requests served.",
		}, []string{"result"})

		galleryLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "gallery_latency_seconds",
			Help:    "Latency distribution for gallery endpoint.",
			Buckets: prometheus.DefBuckets,
		})

		roadmapRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "roadmap_requests_total",
			Help: "Total number of roadmap stage requests served.",
		}, []string{"result"})

		roadmapLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "roadmap_latency_seconds",
			Help:    "Latency distribution for roadmap endpoints.",
			Buckets: prometheus.DefBuckets,
		})

		dashboardRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "student_dashboard_requests_total",
			Help: "Total number of student dashboard requests.",
		}, []string{"result"})

		dashboardLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "student_dashboard_latency_seconds",
			Help:    "Latency distribution for student dashboard endpoint.",
			Buckets: prometheus.DefBuckets,
		})

		contactSubmissions = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "contact_submissions_total",
			Help: "Total number of contact submissions processed.",
		}, []string{"status"})

		uploadRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "upload_requests_total",
			Help: "Total number of upload requests processed by type.",
		}, []string{"type"})

		uploadRejected = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "upload_rejected_total",
			Help: "Total number of upload rejections segmented by reason.",
		}, []string{"reason"})

		uploadLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "upload_latency_seconds",
			Help:    "Latency distribution for upload endpoint.",
			Buckets: prometheus.DefBuckets,
		})

		prometheus.MustRegister(
			adminRequestsTotal,
			adminLatencySeconds,
			adminErrorsTotal,
			chatConnectionsTotal,
			chatDisconnectsTotal,
			chatMessagesSent,
			sseClientsActive,
			notificationsPublishedTotal,
			realtimeErrorsTotal,
			activeActivitiesRequests,
			activeActivitiesLatency,
			announcementsRequests,
			announcementsLatency,
			galleryRequests,
			galleryLatency,
			roadmapRequests,
			roadmapLatency,
			dashboardRequests,
			dashboardLatency,
			contactSubmissions,
			uploadRequests,
			uploadRejected,
			uploadLatency,
		)
	})
}

// AdminRequests exposes the counter for admin requests.
func AdminRequests() *prometheus.CounterVec {
	RegisterMetrics()
	return adminRequestsTotal
}

// AdminLatency exposes the latency histogram for admin requests.
func AdminLatency() *prometheus.HistogramVec {
	RegisterMetrics()
	return adminLatencySeconds
}

// AdminErrors exposes the counter for admin error responses.
func AdminErrors() *prometheus.CounterVec {
	RegisterMetrics()
	return adminErrorsTotal
}

// ChatConnectionsTotal exposes the chat connection counter.
func ChatConnectionsTotal() prometheus.Counter {
	RegisterMetrics()
	return chatConnectionsTotal
}

// ChatDisconnectsTotal exposes the chat disconnect counter.
func ChatDisconnectsTotal() prometheus.Counter {
	RegisterMetrics()
	return chatDisconnectsTotal
}

// ChatMessagesSent exposes the chat messages counter vector.
func ChatMessagesSent() *prometheus.CounterVec {
	RegisterMetrics()
	return chatMessagesSent
}

// SSEClientsActive exposes the gauge tracking active SSE clients.
func SSEClientsActive() prometheus.Gauge {
	RegisterMetrics()
	return sseClientsActive
}

// NotificationsPublishedTotal exposes the notifications counter vector.
func NotificationsPublishedTotal() *prometheus.CounterVec {
	RegisterMetrics()
	return notificationsPublishedTotal
}

// RealtimeErrorsTotal exposes the realtime error counter vector.
func RealtimeErrorsTotal() *prometheus.CounterVec {
	RegisterMetrics()
	return realtimeErrorsTotal
}

// ActiveActivitiesRequests exposes the active activities counter.
func ActiveActivitiesRequests() *prometheus.CounterVec {
	RegisterMetrics()
	return activeActivitiesRequests
}

// ActiveActivitiesLatency exposes the active activities latency histogram.
func ActiveActivitiesLatency() prometheus.Histogram {
	RegisterMetrics()
	return activeActivitiesLatency
}

// AnnouncementsRequests exposes the announcements counter.
func AnnouncementsRequests() *prometheus.CounterVec {
	RegisterMetrics()
	return announcementsRequests
}

// AnnouncementsLatency exposes the announcements latency histogram.
func AnnouncementsLatency() prometheus.Histogram {
	RegisterMetrics()
	return announcementsLatency
}

// GalleryRequests exposes the gallery counter.
func GalleryRequests() *prometheus.CounterVec {
	RegisterMetrics()
	return galleryRequests
}

// GalleryLatency exposes the gallery latency histogram.
func GalleryLatency() prometheus.Histogram {
	RegisterMetrics()
	return galleryLatency
}

// RoadmapRequests exposes roadmap counter.
func RoadmapRequests() *prometheus.CounterVec {
	RegisterMetrics()
	return roadmapRequests
}

// RoadmapLatency exposes roadmap latency histogram.
func RoadmapLatency() prometheus.Histogram {
	RegisterMetrics()
	return roadmapLatency
}

// DashboardRequests exposes dashboard counter.
func DashboardRequests() *prometheus.CounterVec {
	RegisterMetrics()
	return dashboardRequests
}

// DashboardLatency exposes dashboard latency histogram.
func DashboardLatency() prometheus.Histogram {
	RegisterMetrics()
	return dashboardLatency
}

// ContactSubmissions exposes the contact submissions counter.
func ContactSubmissions() *prometheus.CounterVec {
	RegisterMetrics()
	return contactSubmissions
}

// UploadRequests exposes the upload requests counter.
func UploadRequests() *prometheus.CounterVec {
	RegisterMetrics()
	return uploadRequests
}

// UploadRejected exposes the upload rejection counter.
func UploadRejected() *prometheus.CounterVec {
	RegisterMetrics()
	return uploadRejected
}

// UploadLatency exposes the upload latency histogram.
func UploadLatency() prometheus.Histogram {
	RegisterMetrics()
	return uploadLatency
}
