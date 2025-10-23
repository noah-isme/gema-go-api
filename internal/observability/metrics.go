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
	chatMessagesSent            *prometheus.CounterVec
	sseClientsActive            prometheus.Gauge
	notificationsPublishedTotal *prometheus.CounterVec
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

		prometheus.MustRegister(
			adminRequestsTotal,
			adminLatencySeconds,
			adminErrorsTotal,
			chatConnectionsTotal,
			chatMessagesSent,
			sseClientsActive,
			notificationsPublishedTotal,
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
