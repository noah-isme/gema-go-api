package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimit creates a per-user rate limiter middleware instance.
func RateLimit(identifier string, max int, window time.Duration) fiber.Handler {
	if max <= 0 {
		max = 10
	}
	if window <= 0 {
		window = time.Second
	}

	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		KeyGenerator: func(c *fiber.Ctx) string {
			userID := fmt.Sprintf("%v", c.Locals("user_id"))
			if userID == "" || userID == "0" {
				userID = c.IP()
			}
			return fmt.Sprintf("%s:%s", identifier, userID)
		},
	})
}
