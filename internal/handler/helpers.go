package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/noah-isme/gema-go-api/internal/service"
)

func splitAndTrim(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseQueryInt(c *fiber.Ctx, key string) (int, error) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func userIDFromContext(c *fiber.Ctx) uint {
	if v := c.Locals("user_id"); v != nil {
		if id, ok := v.(uint); ok {
			return id
		}
		if id, ok := v.(int); ok {
			if id < 0 {
				return 0
			}
			return uint(id)
		}
	}
	return 0
}

func userRoleFromContext(c *fiber.Ctx) string {
	if v := c.Locals("user_role"); v != nil {
		if role, ok := v.(string); ok {
			return role
		}
	}
	return ""
}

func activityActorFromContext(c *fiber.Ctx) service.ActivityActor {
	return service.ActivityActor{
		ID:   userIDFromContext(c),
		Role: userRoleFromContext(c),
	}
}

func isValidationError(err error) bool {
	var validationErrors validator.ValidationErrors
	return errors.As(err, &validationErrors)
}
