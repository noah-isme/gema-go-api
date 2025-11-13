package utils_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/utils"
)

func TestOKIncludesMetaAndDefaults(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		data := map[string]string{"hello": "world"}
		meta := map[string]int{"page": 1}
		return utils.OK(c, data, "", meta)
	})

	resp := performRequest(t, app, http.MethodGet, "/")
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var payload struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    map[string]string      `json:"data"`
		Meta    map[string]interface{} `json:"meta"`
	}
	decode(t, resp, &payload)

	require.True(t, payload.Success)
	require.Equal(t, "success", payload.Message)
	require.Equal(t, "world", payload.Data["hello"])
	require.Equal(t, float64(1), payload.Meta["page"])
}

func TestFailIncludesDetails(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		details := map[string]string{"field": "studentId"}
		return utils.Fail(c, fiber.StatusBadRequest, "invalid payload", details)
	})

	resp := performRequest(t, app, http.MethodGet, "/")
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var payload struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Details map[string]string      `json:"details"`
		Data    map[string]interface{} `json:"data"`
	}
	decode(t, resp, &payload)

	require.False(t, payload.Success)
	require.Equal(t, "invalid payload", payload.Message)
	require.Equal(t, "studentId", payload.Details["field"])
	require.Nil(t, payload.Data)
}

func performRequest(t *testing.T, app *fiber.App, method, path string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	return resp
}

func decode(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(target))
}
