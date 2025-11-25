package middleware

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/time/rate"
)

// PerformanceMetrics holds performance monitoring data
type PerformanceMetrics struct {
	RequestCount        int64                       `json:"request_count"`
	TotalResponseTime   int64                       `json:"total_response_time_ms"`
	AverageResponseTime float64                     `json:"average_response_time_ms"`
	MinResponseTime     int64                       `json:"min_response_time_ms"`
	MaxResponseTime     int64                       `json:"max_response_time_ms"`
	ActiveConnections   int64                       `json:"active_connections"`
	MemoryUsage         runtime.MemStats            `json:"memory_usage"`
	EndpointMetrics     map[string]*EndpointMetrics `json:"endpoint_metrics"`
	LastUpdated         time.Time                   `json:"last_updated"`
	mu                  sync.RWMutex
}

// EndpointMetrics holds metrics for individual endpoints
type EndpointMetrics struct {
	Path                string    `json:"path"`
	Method              string    `json:"method"`
	RequestCount        int64     `json:"request_count"`
	TotalResponseTime   int64     `json:"total_response_time_ms"`
	AverageResponseTime float64   `json:"average_response_time_ms"`
	MinResponseTime     int64     `json:"min_response_time_ms"`
	MaxResponseTime     int64     `json:"max_response_time_ms"`
	ErrorCount          int64     `json:"error_count"`
	ErrorRate           float64   `json:"error_rate"`
	LastAccessed        time.Time `json:"last_accessed"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int
	SkipPaths         []string
	KeyGenerator      func(*fiber.Ctx) string
	OnLimitReached    func(*fiber.Ctx) error
}

// Global performance metrics instance
var globalMetrics = &PerformanceMetrics{
	EndpointMetrics: make(map[string]*EndpointMetrics),
	LastUpdated:     time.Now(),
}

// PerformanceMonitoring creates a middleware for monitoring request/response times and memory usage
func PerformanceMonitoring() fiber.Handler {
	return func(c *fiber.Ctx) error {
		startTime := time.Now()

		// Increment active connections
		atomic.AddInt64(&globalMetrics.ActiveConnections, 1)
		defer atomic.AddInt64(&globalMetrics.ActiveConnections, -1)

		// Process request
		err := c.Next()

		// Calculate response time
		duration := time.Since(startTime)
		responseTimeMs := duration.Milliseconds()

		// Update global metrics
		updateGlobalMetrics(responseTimeMs)

		// Update endpoint-specific metrics
		endpointKey := c.Method() + ":" + c.Route().Path
		updateEndpointMetrics(endpointKey, c.Method(), c.Route().Path, responseTimeMs, err != nil)

		// Log performance data
		logPerformanceData(c, duration, err)

		return err
	}
}

// MemoryMonitoring creates a middleware for memory usage monitoring
func MemoryMonitoring() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Update memory stats before processing
		updateMemoryStats()

		err := c.Next()

		// Update memory stats after processing
		updateMemoryStats()

		return err
	}
}

// RateLimiting creates a rate limiting middleware
func RateLimiting(config RateLimitConfig) fiber.Handler {
	// Create rate limiter map for different keys (IP addresses, user IDs, etc.)
	limiters := make(map[string]*rate.Limiter)
	var mu sync.RWMutex

	// Default key generator (by IP)
	if config.KeyGenerator == nil {
		config.KeyGenerator = func(c *fiber.Ctx) string {
			return c.IP()
		}
	}

	// Default rate limit handler
	if config.OnLimitReached == nil {
		config.OnLimitReached = func(c *fiber.Ctx) error {
			return utils.ErrorResponse(c, fiber.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
				"Too many requests", map[string]string{
					"retry_after": "60",
				})
		}
	}

	return func(c *fiber.Ctx) error {
		// Skip rate limiting for specified paths
		if shouldSkipPath(c.Path(), config.SkipPaths) {
			return c.Next()
		}

		// Get rate limiter key
		key := config.KeyGenerator(c)

		// Get or create rate limiter for this key
		mu.RLock()
		limiter, exists := limiters[key]
		mu.RUnlock()

		if !exists {
			mu.Lock()
			// Double-check after acquiring write lock
			if limiter, exists = limiters[key]; !exists {
				limiter = rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.BurstSize)
				limiters[key] = limiter
			}
			mu.Unlock()
		}

		// Check if request is allowed
		if !limiter.Allow() {
			// Log rate limit exceeded
			traceID := utils.GetTraceID(c)
			utils.GetLogger().WithTraceID(traceID).WithSource("rate_limiter").Warn(
				"Rate limit exceeded", map[string]interface{}{
					"key":                 key,
					"path":                c.Path(),
					"method":              c.Method(),
					"requests_per_second": config.RequestsPerSecond,
					"burst_size":          config.BurstSize,
				})

			return config.OnLimitReached(c)
		}

		return c.Next()
	}
}

// ConnectionPooling creates a middleware for monitoring connection pool usage
func ConnectionPooling() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Add connection pool monitoring headers
		activeConnections := atomic.LoadInt64(&globalMetrics.ActiveConnections)
		c.Set("X-Connection-Pool-Active", fmt.Sprintf("%d", activeConnections))

		return c.Next()
	}
}

// GetPerformanceMetrics returns current performance metrics
func GetPerformanceMetrics() *PerformanceMetrics {
	// Update memory stats first (without holding the lock)
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	globalMetrics.mu.RLock()
	defer globalMetrics.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := &PerformanceMetrics{
		RequestCount:        atomic.LoadInt64(&globalMetrics.RequestCount),
		TotalResponseTime:   atomic.LoadInt64(&globalMetrics.TotalResponseTime),
		AverageResponseTime: globalMetrics.AverageResponseTime,
		MinResponseTime:     atomic.LoadInt64(&globalMetrics.MinResponseTime),
		MaxResponseTime:     atomic.LoadInt64(&globalMetrics.MaxResponseTime),
		ActiveConnections:   atomic.LoadInt64(&globalMetrics.ActiveConnections),
		MemoryUsage:         memStats,
		EndpointMetrics:     make(map[string]*EndpointMetrics),
		LastUpdated:         time.Now(),
	}

	// Copy endpoint metrics
	for key, endpoint := range globalMetrics.EndpointMetrics {
		metrics.EndpointMetrics[key] = &EndpointMetrics{
			Path:                endpoint.Path,
			Method:              endpoint.Method,
			RequestCount:        atomic.LoadInt64(&endpoint.RequestCount),
			TotalResponseTime:   atomic.LoadInt64(&endpoint.TotalResponseTime),
			AverageResponseTime: endpoint.AverageResponseTime,
			MinResponseTime:     atomic.LoadInt64(&endpoint.MinResponseTime),
			MaxResponseTime:     atomic.LoadInt64(&endpoint.MaxResponseTime),
			ErrorCount:          atomic.LoadInt64(&endpoint.ErrorCount),
			ErrorRate:           endpoint.ErrorRate,
			LastAccessed:        endpoint.LastAccessed,
		}
	}

	return metrics
}

// ResetPerformanceMetrics resets all performance metrics
func ResetPerformanceMetrics() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	atomic.StoreInt64(&globalMetrics.RequestCount, 0)
	atomic.StoreInt64(&globalMetrics.TotalResponseTime, 0)
	globalMetrics.AverageResponseTime = 0
	atomic.StoreInt64(&globalMetrics.MinResponseTime, 0)
	atomic.StoreInt64(&globalMetrics.MaxResponseTime, 0)
	atomic.StoreInt64(&globalMetrics.ActiveConnections, 0)
	globalMetrics.EndpointMetrics = make(map[string]*EndpointMetrics)
	globalMetrics.LastUpdated = time.Now()
}

// Helper functions

// updateGlobalMetrics updates global performance metrics
func updateGlobalMetrics(responseTimeMs int64) {
	atomic.AddInt64(&globalMetrics.RequestCount, 1)
	atomic.AddInt64(&globalMetrics.TotalResponseTime, responseTimeMs)

	// Update min response time
	for {
		current := atomic.LoadInt64(&globalMetrics.MinResponseTime)
		if current == 0 || responseTimeMs < current {
			if atomic.CompareAndSwapInt64(&globalMetrics.MinResponseTime, current, responseTimeMs) {
				break
			}
		} else {
			break
		}
	}

	// Update max response time
	for {
		current := atomic.LoadInt64(&globalMetrics.MaxResponseTime)
		if responseTimeMs > current {
			if atomic.CompareAndSwapInt64(&globalMetrics.MaxResponseTime, current, responseTimeMs) {
				break
			}
		} else {
			break
		}
	}

	// Calculate average response time
	globalMetrics.mu.Lock()
	requestCount := atomic.LoadInt64(&globalMetrics.RequestCount)
	totalResponseTime := atomic.LoadInt64(&globalMetrics.TotalResponseTime)
	if requestCount > 0 {
		globalMetrics.AverageResponseTime = float64(totalResponseTime) / float64(requestCount)
	}
	globalMetrics.LastUpdated = time.Now()
	globalMetrics.mu.Unlock()
}

// updateEndpointMetrics updates endpoint-specific metrics
func updateEndpointMetrics(key, method, path string, responseTimeMs int64, hasError bool) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	endpoint, exists := globalMetrics.EndpointMetrics[key]
	if !exists {
		endpoint = &EndpointMetrics{
			Path:         path,
			Method:       method,
			LastAccessed: time.Now(),
		}
		globalMetrics.EndpointMetrics[key] = endpoint
	}

	atomic.AddInt64(&endpoint.RequestCount, 1)
	atomic.AddInt64(&endpoint.TotalResponseTime, responseTimeMs)

	if hasError {
		atomic.AddInt64(&endpoint.ErrorCount, 1)
	}

	// Update min response time
	current := atomic.LoadInt64(&endpoint.MinResponseTime)
	if current == 0 || responseTimeMs < current {
		atomic.StoreInt64(&endpoint.MinResponseTime, responseTimeMs)
	}

	// Update max response time
	current = atomic.LoadInt64(&endpoint.MaxResponseTime)
	if responseTimeMs > current {
		atomic.StoreInt64(&endpoint.MaxResponseTime, responseTimeMs)
	}

	// Calculate averages
	requestCount := atomic.LoadInt64(&endpoint.RequestCount)
	totalResponseTime := atomic.LoadInt64(&endpoint.TotalResponseTime)
	errorCount := atomic.LoadInt64(&endpoint.ErrorCount)

	if requestCount > 0 {
		endpoint.AverageResponseTime = float64(totalResponseTime) / float64(requestCount)
		endpoint.ErrorRate = float64(errorCount) / float64(requestCount) * 100
	}

	endpoint.LastAccessed = time.Now()
}

// updateMemoryStats updates memory usage statistics
func updateMemoryStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	globalMetrics.mu.Lock()
	globalMetrics.MemoryUsage = memStats
	globalMetrics.mu.Unlock()
}

// logPerformanceData logs performance information
func logPerformanceData(c *fiber.Ctx, duration time.Duration, err error) {
	traceID := utils.GetTraceID(c)

	context := map[string]interface{}{
		"method":        c.Method(),
		"path":          c.Path(),
		"duration_ms":   duration.Milliseconds(),
		"status_code":   c.Response().StatusCode(),
		"response_size": len(c.Response().Body()),
	}

	if err != nil {
		context["error"] = err.Error()
	}

	// Log slow requests (> 1 second)
	if duration > time.Second {
		utils.GetLogger().WithTraceID(traceID).WithSource("performance").Warn(
			"Slow request detected", context)
	} else if duration > 500*time.Millisecond {
		utils.GetLogger().WithTraceID(traceID).WithSource("performance").Info(
			"Request performance", context)
	}
}
