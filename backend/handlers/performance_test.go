package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/middleware"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPerformanceHandler(t *testing.T) {
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.logger)
}

func TestGetPerformanceMetrics(t *testing.T) {
	// Reset metrics before test
	middleware.ResetPerformanceMetrics()

	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/metrics", handler.GetPerformanceMetrics)

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetMemoryStats(t *testing.T) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/memory", handler.GetMemoryStats)

	req := httptest.NewRequest("GET", "/memory", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetConnectionPoolStats(t *testing.T) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/pools", handler.GetConnectionPoolStats)

	req := httptest.NewRequest("GET", "/pools", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestResetPerformanceMetrics(t *testing.T) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Post("/reset", handler.ResetPerformanceMetrics)

	req := httptest.NewRequest("POST", "/reset", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestTriggerGC(t *testing.T) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Post("/gc", handler.TriggerGC)

	req := httptest.NewRequest("POST", "/gc", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetSystemInfo(t *testing.T) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/system", handler.GetSystemInfo)

	req := httptest.NewRequest("GET", "/system", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetEndpointMetrics(t *testing.T) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/endpoint", handler.GetEndpointMetrics)

	// Test missing path parameter
	req := httptest.NewRequest("GET", "/endpoint", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// Test with path parameter but no metrics
	req = httptest.NewRequest("GET", "/endpoint?path=/test", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestGetTopEndpoints(t *testing.T) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/top", handler.GetTopEndpoints)

	// Test default sorting (by request count)
	req := httptest.NewRequest("GET", "/top", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test sorting by average response time
	req = httptest.NewRequest("GET", "/top?sort_by=average_response_time", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test sorting by error rate
	req = httptest.NewRequest("GET", "/top?sort_by=error_rate", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test with custom limit
	req = httptest.NewRequest("GET", "/top?limit=5", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test with invalid limit (should use default)
	req = httptest.NewRequest("GET", "/top?limit=0", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test with limit too high (should cap at 100)
	req = httptest.NewRequest("GET", "/top?limit=200", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHealthCheck(t *testing.T) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/health", handler.HealthCheck)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestPerformanceHandlerWithMetrics(t *testing.T) {
	// Reset metrics before test
	middleware.ResetPerformanceMetrics()

	// Create app with performance monitoring
	app := fiber.New()
	app.Use(middleware.PerformanceMonitoring())

	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	// Add test endpoint
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Add performance endpoints
	app.Get("/metrics", handler.GetPerformanceMetrics)
	app.Get("/endpoint", handler.GetEndpointMetrics)

	// Make some requests to generate metrics
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		resp.Body.Close()
	}

	// Check metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Check endpoint-specific metrics
	req = httptest.NewRequest("GET", "/endpoint?path=/test&method=GET", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func BenchmarkGetPerformanceMetrics(b *testing.B) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/metrics", handler.GetPerformanceMetrics)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/metrics", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkGetMemoryStats(b *testing.B) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/memory", handler.GetMemoryStats)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/memory", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkGetSystemInfo(b *testing.B) {
	app := fiber.New()
	logger := utils.GetLogger()
	handler := NewPerformanceHandler(logger)

	app.Get("/system", handler.GetSystemInfo)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/system", nil)
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}
