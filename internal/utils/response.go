package utils

import "github.com/gofiber/fiber/v2"

// APIResponse describes the common structure for API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message"`
}

// SendSuccess sends a successful JSON response with a message.
func SendSuccess(c *fiber.Ctx, message string, data interface{}) error {
	if message == "" {
		message = "success"
	}

	return c.Status(fiber.StatusOK).JSON(APIResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// SendError sends an error JSON response with the given status code.
func SendError(c *fiber.Ctx, status int, message string) error {
	if message == "" {
		message = "error"
	}

	return c.Status(status).JSON(APIResponse{
		Success: false,
		Message: message,
	})
}
