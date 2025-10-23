package observability

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler exposes the Prometheus scrape endpoint via Fiber.
func MetricsHandler() fiber.Handler {
	RegisterMetrics()
	return adaptor.HTTPHandler(promhttp.Handler())
}
