package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestDebugHandler_GetConfig(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:                        "8080",
		Host:                        "localhost",
		Environment:                 "development",
		OpenAIAPIKey:                "sk-test1234567890",
		FrontendURL:                 "http://localhost:3000",
		WSEndpoint:                  "/ws",
		LogLevel:                    "info",
		LogFormat:                   "json",
		EnableAIFeatures:            true,
		EnableWebSocket:             true,
		EnablePerformanceMonitoring: true,
		EnableRateLimiting:          true,
		EnableCircuitBreaker:        true,
		EnableDetailedErrors:        false,
		EnableDebugEndpoints:        true,
	}

	handler := NewDebugHandler(cfg)
	app := fiber.New()
	app.Get("/debug/config", handler.GetConfig)

	// Test
	req := httptest.NewRequest("GET", "/debug/config", nil)
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	assert.True(t, result["success"].(bool))
	assert.NotNil(t, result["data"])

	// Check that sensitive values are masked
	data := result["data"].(map[string]interface{})
	openai := data["openai"].(map[string]interface{})
	assert.Contains(t, openai["api_key"].(string), "****")
}

func TestDebugHandler_GetRoutes(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	handler := NewDebugHandler(cfg)
	app := fiber.New()
	app.Get("/debug/routes", handler.GetRoutes)
	app.Get("/test", func(c *fiber.Ctx) error { return nil })

	// Test
	req := httptest.NewRequest("GET", "/debug/routes", nil)
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	assert.True(t, result["success"].(bool))
	data := result["data"].(map[string]interface{})
	assert.Greater(t, int(data["total"].(float64)), 0)
}

func TestDebugHandler_GetEnvironment(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	handler := NewDebugHandler(cfg)
	app := fiber.New()
	app.Get("/debug/env", handler.GetEnvironment)

	// Set test environment variable
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("TEST_API_KEY", "secret_key")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("TEST_API_KEY")

	// Test
	req := httptest.NewRequest("GET", "/debug/env", nil)
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	assert.True(t, result["success"].(bool))
	data := result["data"].(map[string]interface{})
	env := data["environment"].(map[string]interface{})

	// Check that sensitive values are masked
	if apiKey, ok := env["TEST_API_KEY"]; ok {
		assert.Contains(t, apiKey.(string), "****")
	}
}

func TestDebugHandler_GetSystemInfo(t *testing.T) {
	// Setup
	cfg := &config.Config{}
	handler := NewDebugHandler(cfg)
	app := fiber.New()
	app.Get("/debug/system", handler.GetSystemInfo)

	// Test
	req := httptest.NewRequest("GET", "/debug/system", nil)
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	assert.True(t, result["success"].(bool))
	data := result["data"].(map[string]interface{})
	assert.NotEmpty(t, data["go_version"])
	assert.NotEmpty(t, data["os"])
	assert.NotEmpty(t, data["arch"])
}

func TestDebugHandler_GetFeatureToggles(t *testing.T) {
	// Setup
	cfg := &config.Config{
		EnableAIFeatures:            true,
		EnableWebSocket:             false,
		EnablePerformanceMonitoring: true,
		EnableRateLimiting:          true,
		EnableCircuitBreaker:        false,
		EnableDetailedErrors:        true,
		EnableDebugEndpoints:        true,
	}

	handler := NewDebugHandler(cfg)
	app := fiber.New()
	app.Get("/debug/features", handler.GetFeatureToggles)

	// Test
	req := httptest.NewRequest("GET", "/debug/features", nil)
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	assert.True(t, result["success"].(bool))
	data := result["data"].(map[string]interface{})

	// Check feature toggle values
	aiFeatures := data["ai_features"].(map[string]interface{})
	assert.True(t, aiFeatures["enabled"].(bool))

	websocket := data["websocket"].(map[string]interface{})
	assert.False(t, websocket["enabled"].(bool))
}

func TestDebugHandler_GetHealthChecks(t *testing.T) {
	// Setup
	cfg := &config.Config{
		OpenAIAPIKey:     "sk-test1234567890",
		EnableAIFeatures: true,
	}

	handler := NewDebugHandler(cfg)
	app := fiber.New()
	app.Get("/debug/health", handler.GetHealthChecks)

	// Test
	req := httptest.NewRequest("GET", "/debug/health", nil)
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	assert.True(t, result["success"].(bool))
	data := result["data"].(map[string]interface{})
	assert.NotNil(t, data["server"])
	assert.NotNil(t, data["configuration"])
	assert.NotNil(t, data["openai"])
}

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "short string",
			input:    "short",
			expected: "****",
		},
		{
			name:     "long string",
			input:    "sk-1234567890abcdef",
			expected: "sk-1****cdef",
		},
		{
			name:     "api key",
			input:    "sk-proj-1234567890abcdefghijklmnop",
			expected: "sk-p****mnop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
