package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/observability"
)

// Observability attaches Prometheus metrics and structured latency/error logging for admin endpoints.
func Observability(logger zerolog.Logger) fiber.Handler {
	observability.RegisterMetrics()

	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		if strings.HasPrefix(c.Path(), "/api/admin") {
			route := routeTemplate(c)
			method := c.Method()
			status := c.Response().StatusCode()
			statusLabel := fmt.Sprintf("%d", status)

			observability.AdminRequests().WithLabelValues(method, route, statusLabel).Inc()
			observability.AdminLatency().WithLabelValues(method, route).Observe(duration.Seconds())
			if status >= fiber.StatusBadRequest {
				observability.AdminErrors().WithLabelValues(method, route, statusLabel).Inc()
			}

			latencyMs := float64(duration) / float64(time.Millisecond)
			bucket := latencyBucket(duration)
			requestLogger := logger.With().
				Str("correlation_id", GetCorrelationID(c)).
				Str("route", route).
				Str("method", method).
				Int("status", status).
				Float64("latency_ms", latencyMs).
				Str("latency_bucket", bucket).
				Logger()

			switch {
			case status >= fiber.StatusInternalServerError:
				requestLogger.Error().Msg("admin request failed")
			case status >= fiber.StatusBadRequest:
				requestLogger.Warn().Msg("admin request completed with client error")
			default:
				requestLogger.Info().Msg("admin request completed")
			}
		}

		return err
	}
}

func routeTemplate(c *fiber.Ctx) string {
	if c.Route() != nil && c.Route().Path != "" {
		return c.Route().Path
	}
	return c.Path()
}

func latencyBucket(duration time.Duration) string {
	switch {
	case duration <= 25*time.Millisecond:
		return "<=25ms"
	case duration <= 50*time.Millisecond:
		return "<=50ms"
	case duration <= 100*time.Millisecond:
		return "<=100ms"
	case duration <= 250*time.Millisecond:
		return "<=250ms"
	case duration <= 500*time.Millisecond:
		return "<=500ms"
	default:
		return ">500ms"
	}
}
