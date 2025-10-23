package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/noah-isme/gema-go-api/internal/utils"
)

// RequireRole ensures that the authenticated user possesses one of the allowed roles.
func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}

	return func(c *fiber.Ctx) error {
		roleValue := c.Locals("user_role")
		role := normalizeRoleValue(roleValue)
		if _, ok := allowed[role]; !ok {
			return utils.SendError(c, fiber.StatusForbidden, "insufficient permissions")
		}
		return c.Next()
	}
}

func normalizeRoleValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.ToLower(strings.TrimSpace(v))
	case fmt.Stringer:
		return strings.ToLower(strings.TrimSpace(v.String()))
	default:
		if value == nil {
			return ""
		}
		return strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", value)))
	}
}
