package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/noah-isme/gema-go-api/internal/utils"
)

// Auth role constants used by WithAuth helper.
const (
	AuthRoleAny     = "any"
	AuthRoleAdmin   = "admin"
	AuthRoleStudent = "student"
)

// AuthOptions configures the WithAuth helper.
type AuthOptions struct {
	Role        string
	RequireUser bool
}

// WithAuth wraps a handler with basic authentication/authorization guards.
func WithAuth(handler fiber.Handler, opts AuthOptions) fiber.Handler {
	role := strings.ToLower(strings.TrimSpace(opts.Role))
	if role == "" {
		role = AuthRoleAny
	}

	requireUser := opts.RequireUser
	if !requireUser && role != AuthRoleAny {
		requireUser = true
	}

	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		if requireUser && userID == nil {
			return utils.Fail(c, fiber.StatusUnauthorized, "authentication required", nil)
		}

		if role == AuthRoleAny {
			// Allow anonymous access when RequireUser=false; otherwise userID must exist.
			if !requireUser || userID != nil {
				return handler(c)
			}
			return utils.Fail(c, fiber.StatusUnauthorized, "authentication required", nil)
		}

		currentRole := normalizeRoleValue(c.Locals("user_role"))
		switch role {
		case AuthRoleStudent:
			if currentRole != "student" {
				return utils.Fail(c, fiber.StatusForbidden, "insufficient permissions", nil)
			}
		case AuthRoleAdmin:
			if currentRole != "admin" && currentRole != "teacher" {
				return utils.Fail(c, fiber.StatusForbidden, "insufficient permissions", nil)
			}
		default:
			if currentRole != role {
				return utils.Fail(c, fiber.StatusForbidden, "insufficient permissions", nil)
			}
		}

		return handler(c)
	}
}
