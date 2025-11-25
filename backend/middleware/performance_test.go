package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceMonitoring(t *testing.T) {
	// Reset metrics before test
	ResetPerformanceMetrics()

	app := fiber.New()
	app.Use(PerformanceMonitoring())

	app.Get("/test", func(c *fiber.Ctx) error {
		time.Sleep(10 * time.Millisecond) // Simulate processing time
		return c.SendString("OK")
	})

	// Make a test request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Check metrics
	metrics := GetPerformanceMetrics()
	assert.Equal(t, int64(1), metrics.RequestCount)
	assert.True(t, metrics.AverageResponseTime > 0)
	assert.True(t, metrics.MinResponseTime > 0)
	assert.True(t, metrics.MaxResponseTime > 0)
	assert.Equal(t, int64(0), metrics.ActiveConnections) // Should be 0 after request completes
}

func TestMemoryMonitoring(t *testing.T) {
	app := fiber.New()
	app.Use(MemoryMonitoring())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Check that memory stats are updated
	metrics := GetPerformanceMetrics()
	assert.True(t, metrics.MemoryUsage.Alloc > 0)
}

func TestRateLimiting(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 2,
		BurstSize:         2, // Allow 2 requests in burst
	}

	app := fiber.New()
	app.Use(RateLimiting(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// First request should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Second request should succeed (within burst)
	req = httptest.NewRequest("GET", "/test", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Third request should be rate limited (burst exhausted)
	req = httptest.NewRequest("GET", "/test", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)
}

func TestConnectionPooling(t *testing.T) {
	app := fiber.New()
	app.Use(ConnectionPooling())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Check that connection pool header is set
	assert.NotEmpty(t, resp.Header.Get("X-Connection-Pool-Active"))
}

func TestGetPerformanceMetrics(t *testing.T) {
	// Reset metrics before test
	ResetPerformanceMetrics()

	app := fiber.New()
	app.Use(PerformanceMonitoring())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Make multiple requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}

	metrics := GetPerformanceMetrics()
	assert.Equal(t, int64(5), metrics.RequestCount)
	assert.True(t, metrics.AverageResponseTime >= 0) // Allow 0 for very fast responses
	assert.Len(t, metrics.EndpointMetrics, 1)

	// Check endpoint-specific metrics
	endpointKey := "GET:/test"
	endpointMetrics, exists := metrics.EndpointMetrics[endpointKey]
	require.True(t, exists)
	assert.Equal(t, int64(5), endpointMetrics.RequestCount)
	assert.Equal(t, "GET", endpointMetrics.Method)
	assert.Equal(t, "/test", endpointMetrics.Path)
	assert.Equal(t, float64(0), endpointMetrics.ErrorRate) // No errors
}

func TestResetPerformanceMetrics(t *testing.T) {
	// Reset metrics before test
	ResetPerformanceMetrics()

	app := fiber.New()
	app.Use(PerformanceMonitoring())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Make a request to generate metrics
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify metrics exist
	metrics := GetPerformanceMetrics()
	assert.Equal(t, int64(1), metrics.RequestCount)

	// Reset metrics
	ResetPerformanceMetrics()

	// Verify metrics are reset
	metrics = GetPerformanceMetrics()
	assert.Equal(t, int64(0), metrics.RequestCount)
	assert.Equal(t, float64(0), metrics.AverageResponseTime)
	assert.Len(t, metrics.EndpointMetrics, 0)
}

func TestRateLimitingWithCustomKeyGenerator(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 1,
		BurstSize:         1,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.Get("X-User-ID", "anonymous")
		},
	}

	app := fiber.New()
	app.Use(RateLimiting(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Request with User ID 1
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "user1")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Request with User ID 2 (different user, should succeed)
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "user2")
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Another request with User ID 1 (should be rate limited)
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "user1")
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)
}

func TestRateLimitingSkipPaths(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 1,
		BurstSize:         1,
		SkipPaths:         []string{"/health"},
	}

	app := fiber.New()
	app.Use(RateLimiting(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("Healthy")
	})

	// Multiple requests to /health should all succeed (skipped from rate limiting)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}

	// First request to /test should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Second request to /test should be rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)
}

func BenchmarkPerformanceMonitoring(b *testing.B) {
	app := fiber.New()
	app.Use(PerformanceMonitoring())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			_, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRateLimiting(b *testing.B) {
	config := RateLimitConfig{
		RequestsPerSecond: 1000, // High limit for benchmarking
		BurstSize:         100,
	}

	app := fiber.New()
	app.Use(RateLimiting(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			_, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
