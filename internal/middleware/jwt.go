package middleware

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/noah-isme/gema-go-api/internal/utils"
)

// JWTProtected returns a middleware that validates JWT bearer tokens.
func JWTProtected(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authorization := c.Get("Authorization")
		if authorization == "" {
			return utils.SendError(c, fiber.StatusUnauthorized, "authorization header missing")
		}

		const bearer = "Bearer "
		if !strings.HasPrefix(strings.ToLower(authorization), strings.ToLower(bearer)) {
			return utils.SendError(c, fiber.StatusUnauthorized, "invalid authorization header")
		}

		tokenString := strings.TrimSpace(authorization[len(bearer):])
		if tokenString == "" {
			return utils.SendError(c, fiber.StatusUnauthorized, "invalid token")
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			return utils.SendError(c, fiber.StatusUnauthorized, "invalid token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, "invalid token claims")
		}

		if userID := extractUserIDFromClaims(claims); userID != nil {
			c.Locals("user_id", *userID)
		}

		return c.Next()
	}
}

func extractUserIDFromClaims(claims jwt.MapClaims) *uint {
	keys := []string{"sub", "user_id", "id"}
	for _, key := range keys {
		if value, ok := claims[key]; ok {
			if normalized, err := normalizeUserID(value); err == nil {
				return &normalized
			}
		}
	}

	return nil
}

func normalizeUserID(value interface{}) (uint, error) {
	switch v := value.(type) {
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("invalid subject")
		}
		return uint(v), nil
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return uint(parsed), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("invalid subject")
		}
		return uint(v), nil
	default:
		return 0, fmt.Errorf("unsupported subject type")
	}
}
