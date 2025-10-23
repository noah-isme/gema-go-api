package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type correlationIDKey struct{}

var correlationKey = correlationIDKey{}

// CorrelationID middleware ensures every request carries a correlation identifier for tracing across services.
func CorrelationID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		incoming := strings.TrimSpace(c.Get("X-Correlation-ID"))
		if incoming == "" {
			incoming = strings.TrimSpace(c.Get("X-Request-ID"))
		}
		if incoming == "" {
			incoming = uuid.NewString()
		}

		c.Locals("correlation_id", incoming)
		c.Set("X-Correlation-ID", incoming)

		ctx := context.WithValue(c.Context(), correlationKey, incoming)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// CorrelationIDFromContext extracts the correlation identifier from context, if present.
func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value := ctx.Value(correlationKey); value != nil {
		if id, ok := value.(string); ok {
			return id
		}
	}
	return ""
}

// GetCorrelationID returns the correlation identifier bound to the active request.
func GetCorrelationID(c *fiber.Ctx) string {
	if c == nil {
		return ""
	}
	if value := c.Locals("correlation_id"); value != nil {
		if id, ok := value.(string); ok {
			return id
		}
	}
	return CorrelationIDFromContext(c.Context())
}

// ContextWithCorrelation attaches the correlation identifier to the provided context.
func ContextWithCorrelation(ctx context.Context, correlationID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(correlationID) == "" {
		return ctx
	}
	return context.WithValue(ctx, correlationKey, strings.TrimSpace(correlationID))
}
