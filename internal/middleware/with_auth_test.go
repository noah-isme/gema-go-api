package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/middleware"
)

func TestWithAuthStudentRole(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(10))
		c.Locals("user_role", "Student")
		return c.Next()
	})
	app.Get("/", middleware.WithAuth(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	}, middleware.AuthOptions{Role: middleware.AuthRoleStudent}))

	resp := perform(t, app)
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestWithAuthStudentRoleDenied(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(10))
		c.Locals("user_role", "guest")
		return c.Next()
	})
	app.Get("/", middleware.WithAuth(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	}, middleware.AuthOptions{Role: middleware.AuthRoleStudent}))

	resp := perform(t, app)
	require.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestWithAuthAdminAllowsTeacher(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(1))
		c.Locals("user_role", "teacher")
		return c.Next()
	})
	app.Get("/", middleware.WithAuth(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}, middleware.AuthOptions{Role: middleware.AuthRoleAdmin}))

	resp := perform(t, app)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestWithAuthAnyRequiresUserByDefault(t *testing.T) {
	app := fiber.New()
	app.Get("/", middleware.WithAuth(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}, middleware.AuthOptions{Role: middleware.AuthRoleAny}))

	resp := perform(t, app)
	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestWithAuthAnyAllowsAnonymousWhenOptedIn(t *testing.T) {
	app := fiber.New()
	app.Get("/", middleware.WithAuth(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}, middleware.AuthOptions{Role: middleware.AuthRoleAny, RequireUser: false}))

	resp := perform(t, app)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func perform(t *testing.T, app *fiber.App) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	return resp
}
