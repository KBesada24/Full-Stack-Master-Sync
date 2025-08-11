package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateFiberApp tests the Fiber app creation
func TestCreateFiberApp(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
	}
	logger := utils.GetLogger()
	recoveryService := utils.NewErrorRecoveryService(logger)

	app := createFiberApp(cfg, logger, recoveryService)

	assert.NotNil(t, app)
	assert.Equal(t, "Full Stack Master Sync Backend v1.0.0", app.Config().AppName)
}

// TestHealthCheckHandler tests the health check endpoint
func TestHealthCheckHandler(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
	}

	app := fiber.New()
	app.Get("/health", healthCheckHandler(cfg))

	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Check that response contains expected fields
	assert.Contains(t, string(body), "success")
	assert.Contains(t, string(body), "healthy")
	assert.Contains(t, string(body), "version")
	assert.Contains(t, string(body), "environment")
}

// TestSetupMiddleware tests middleware setup
func TestSetupMiddleware(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
		FrontendURL: "http://localhost:3000",
	}
	logger := utils.GetLogger()
	recoveryService := utils.NewErrorRecoveryService(logger)

	app := fiber.New()
	setupMiddleware(app, cfg, logger, recoveryService)

	// Test that middleware is properly configured by making a request
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"test": "ok"})
	})

	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check that middleware is working (status should be OK)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check correlation ID headers are present
	assert.NotEmpty(t, resp.Header.Get("X-Trace-ID"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

// TestSetupRoutes tests route setup
func TestSetupRoutes(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
	}
	logger := utils.GetLogger()
	recoveryService := utils.NewErrorRecoveryService(logger)

	app := fiber.New()
	setupRoutes(app, cfg, logger, recoveryService)

	// Test health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test API info endpoint
	req, err = http.NewRequest("GET", "/api", nil)
	require.NoError(t, err)

	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "Full Stack Master Sync API")
	assert.Contains(t, string(body), "version")
	assert.Contains(t, string(body), "endpoints")
}

// TestErrorHandler tests the custom error handler
func TestErrorHandler(t *testing.T) {
	logger := utils.GetLogger()
	recoveryService := utils.NewErrorRecoveryService(logger)
	errorHandler := createErrorHandler(logger, recoveryService)

	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})

	// Create a route that returns an error
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "Test error")
	})

	req, err := http.NewRequest("GET", "/error", nil)
	require.NoError(t, err)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "error")
	assert.Contains(t, string(body), "REQUEST_ERROR")
	assert.Contains(t, string(body), "trace_id")
}

// TestServerConfiguration tests server configuration
func TestServerConfiguration(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
		FrontendURL: "http://localhost:3000",
	}
	logger := utils.GetLogger()
	recoveryService := utils.NewErrorRecoveryService(logger)

	app := createFiberApp(cfg, logger, recoveryService)
	setupMiddleware(app, cfg, logger, recoveryService)
	setupRoutes(app, cfg, logger, recoveryService)

	// Test that the app is properly configured
	assert.NotNil(t, app)

	// Test a complete request flow
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify response structure
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "success")
	assert.Contains(t, string(body), "data")
	assert.Contains(t, string(body), "timestamp")
	assert.Contains(t, string(body), "trace_id")
}

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
	}

	app := fiber.New()
	app.Get("/health", healthCheckHandler(cfg))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		resp, _ := app.Test(req, -1)
		resp.Body.Close()
	}
}
