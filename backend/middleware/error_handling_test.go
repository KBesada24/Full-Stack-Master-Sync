package middleware

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnhancedErrorHandlingMiddleware_Success(t *testing.T) {
	app := fiber.New()
	app.Use(EnhancedErrorHandlingMiddleware())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestEnhancedErrorHandlingMiddleware_PanicRecovery(t *testing.T) {
	config := &EnhancedErrorHandlingConfig{
		EnableStackTrace:     true,
		EnableDetailedErrors: true,
		Logger:               utils.NewLogger("debug", "json"),
		RecoveryService:      utils.NewErrorRecoveryService(nil),
	}

	app := fiber.New()
	app.Use(EnhancedErrorHandlingMiddleware(config))

	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "INTERNAL_SERVER_ERROR")
	assert.Contains(t, bodyStr, "An unexpected error occurred")
}

func TestEnhancedErrorHandlingMiddleware_FiberError(t *testing.T) {
	app := fiber.New()
	app.Use(EnhancedErrorHandlingMiddleware())

	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request error")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "BAD_REQUEST")
	assert.Contains(t, bodyStr, "Bad request error")
}

func TestEnhancedErrorHandlingMiddleware_CircuitBreakerError(t *testing.T) {
	app := fiber.New()
	app.Use(EnhancedErrorHandlingMiddleware())

	app.Get("/circuit-breaker", func(c *fiber.Ctx) error {
		return &utils.CircuitBreakerError{
			State:   utils.StateOpen,
			Message: "circuit breaker is open",
		}
	})

	req := httptest.NewRequest("GET", "/circuit-breaker", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "CIRCUIT_BREAKER_OPEN")
	assert.Contains(t, bodyStr, "circuit_breaker")
}

func TestEnhancedErrorHandlingMiddleware_RetryableError(t *testing.T) {
	app := fiber.New()
	app.Use(EnhancedErrorHandlingMiddleware())

	app.Get("/retry-error", func(c *fiber.Ctx) error {
		return &utils.RetryableError{
			Err:       errors.New("connection timeout"),
			Retryable: true,
			Attempt:   3,
		}
	})

	req := httptest.NewRequest("GET", "/retry-error", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "RETRY_EXHAUSTED")
	assert.Contains(t, bodyStr, "retryable")
}

func TestEnhancedErrorHandlingMiddleware_ContextErrors(t *testing.T) {
	testCases := []struct {
		name           string
		error          error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "context canceled",
			error:          context.Canceled,
			expectedStatus: fiber.StatusRequestTimeout,
			expectedCode:   "REQUEST_CANCELLED",
		},
		{
			name:           "context deadline exceeded",
			error:          context.DeadlineExceeded,
			expectedStatus: fiber.StatusRequestTimeout,
			expectedCode:   "REQUEST_TIMEOUT",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(EnhancedErrorHandlingMiddleware())

			app.Get("/test", func(c *fiber.Ctx) error {
				return tc.error
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			bodyStr := string(body)
			assert.Contains(t, bodyStr, tc.expectedCode)
		})
	}
}

func TestEnhancedErrorHandlingMiddleware_NetworkErrors(t *testing.T) {
	testCases := []struct {
		name           string
		errorMessage   string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "connection refused",
			errorMessage:   "connection refused",
			expectedStatus: fiber.StatusBadGateway,
			expectedCode:   "NETWORK_ERROR",
		},
		{
			name:           "timeout error",
			errorMessage:   "request timeout",
			expectedStatus: fiber.StatusGatewayTimeout,
			expectedCode:   "TIMEOUT_ERROR",
		},
		{
			name:           "rate limit error",
			errorMessage:   "rate limit exceeded",
			expectedStatus: fiber.StatusTooManyRequests,
			expectedCode:   "RATE_LIMIT_EXCEEDED",
		},
		{
			name:           "authentication error",
			errorMessage:   "authentication failed",
			expectedStatus: fiber.StatusUnauthorized,
			expectedCode:   "AUTHENTICATION_FAILED",
		},
		{
			name:           "authorization error",
			errorMessage:   "access denied",
			expectedStatus: fiber.StatusForbidden,
			expectedCode:   "AUTHORIZATION_FAILED",
		},
		{
			name:           "validation error",
			errorMessage:   "validation failed",
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "not found error",
			errorMessage:   "resource not found",
			expectedStatus: fiber.StatusNotFound,
			expectedCode:   "RESOURCE_NOT_FOUND",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(EnhancedErrorHandlingMiddleware())

			app.Get("/test", func(c *fiber.Ctx) error {
				return errors.New(tc.errorMessage)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			bodyStr := string(body)
			assert.Contains(t, bodyStr, tc.expectedCode)
		})
	}
}

func TestEnhancedErrorHandlingMiddleware_DetailedErrors(t *testing.T) {
	config := &EnhancedErrorHandlingConfig{
		EnableDetailedErrors: true,
		Logger:               utils.NewLogger("debug", "json"),
	}

	app := fiber.New()
	app.Use(EnhancedErrorHandlingMiddleware(config))

	testError := errors.New("detailed test error")
	app.Get("/test", func(c *fiber.Ctx) error {
		return testError
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "detailed test error")
	assert.Contains(t, bodyStr, "original_error")
	assert.Contains(t, bodyStr, "error_type")
}

func TestEnhancedErrorHandlingMiddleware_DisabledDetailedErrors(t *testing.T) {
	config := &EnhancedErrorHandlingConfig{
		EnableDetailedErrors: false,
		Logger:               utils.NewLogger("debug", "json"),
	}

	app := fiber.New()
	app.Use(EnhancedErrorHandlingMiddleware(config))

	testError := errors.New("detailed test error")
	app.Get("/test", func(c *fiber.Ctx) error {
		return testError
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.NotContains(t, bodyStr, "detailed test error")
	assert.NotContains(t, bodyStr, "original_error")
}

func TestHealthCheckErrorHandler_Success(t *testing.T) {
	recoveryService := utils.NewErrorRecoveryService(nil)
	recoveryService.RegisterHealthCheck("test", func(ctx context.Context) error {
		return nil
	})

	app := fiber.New()
	app.Get("/health", HealthCheckErrorHandler(recoveryService))

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "Health checks passed")
}

func TestHealthCheckErrorHandler_Failure(t *testing.T) {
	recoveryService := utils.NewErrorRecoveryService(nil)
	recoveryService.RegisterHealthCheck("failing", func(ctx context.Context) error {
		return errors.New("health check failed")
	})

	app := fiber.New()
	app.Get("/health", HealthCheckErrorHandler(recoveryService))

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "Health check failures detected")
}

func TestCategorizeError_FiberErrors(t *testing.T) {
	testCases := []struct {
		statusCode   int
		expectedCode string
	}{
		{fiber.StatusBadRequest, "BAD_REQUEST"},
		{fiber.StatusUnauthorized, "UNAUTHORIZED"},
		{fiber.StatusForbidden, "FORBIDDEN"},
		{fiber.StatusNotFound, "NOT_FOUND"},
		{fiber.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
		{fiber.StatusRequestTimeout, "REQUEST_TIMEOUT"},
		{fiber.StatusTooManyRequests, "TOO_MANY_REQUESTS"},
		{fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR"},
		{fiber.StatusBadGateway, "BAD_GATEWAY"},
		{fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
		{fiber.StatusGatewayTimeout, "GATEWAY_TIMEOUT"},
		{999, "UNKNOWN_ERROR"}, // Unknown status code
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("status_%d", tc.statusCode), func(t *testing.T) {
			fiberErr := fiber.NewError(tc.statusCode, "test message")
			statusCode, errorCode, _ := categorizeError(fiberErr)

			assert.Equal(t, tc.statusCode, statusCode)
			assert.Equal(t, tc.expectedCode, errorCode)
		})
	}
}

func TestContainsAnyPattern(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		patterns []string
		expected bool
	}{
		{
			name:     "match found",
			text:     "Connection refused by server",
			patterns: []string{"connection refused", "timeout"},
			expected: true,
		},
		{
			name:     "case insensitive match",
			text:     "CONNECTION REFUSED by server",
			patterns: []string{"connection refused", "timeout"},
			expected: true,
		},
		{
			name:     "no match",
			text:     "Everything is fine",
			patterns: []string{"connection refused", "timeout"},
			expected: false,
		},
		{
			name:     "empty patterns",
			text:     "Some text",
			patterns: []string{},
			expected: false,
		},
		{
			name:     "empty text",
			text:     "",
			patterns: []string{"pattern"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsAnyPattern(tc.text, tc.patterns)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreateEnhancedErrorResponse(t *testing.T) {
	code := "TEST_ERROR"
	message := "Test error message"
	traceID := "test-trace-id"
	config := DefaultEnhancedErrorHandlingConfig()
	details := map[string]string{"key": "value"}

	errorResponse := createEnhancedErrorResponse(code, message, traceID, config, details)

	assert.Equal(t, code, errorResponse.Code)
	assert.Equal(t, message, errorResponse.Message)
	assert.Equal(t, "value", errorResponse.Details["key"])
	assert.Equal(t, traceID, errorResponse.Details["correlation_id"])
	assert.Contains(t, errorResponse.Details, "timestamp")
}

func TestCreateEnhancedErrorResponse_EmptyTraceID(t *testing.T) {
	code := "TEST_ERROR"
	message := "Test error message"
	traceID := ""
	config := DefaultEnhancedErrorHandlingConfig()

	errorResponse := createEnhancedErrorResponse(code, message, traceID, config, nil)

	assert.Equal(t, code, errorResponse.Code)
	assert.Equal(t, message, errorResponse.Message)
	assert.NotEmpty(t, errorResponse.Details["correlation_id"])
	assert.NotEqual(t, traceID, errorResponse.Details["correlation_id"])
}

func BenchmarkEnhancedErrorHandlingMiddleware(b *testing.B) {
	app := fiber.New()
	app.Use(EnhancedErrorHandlingMiddleware())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		app.Test(req)
	}
}

func BenchmarkEnhancedErrorHandlingMiddleware_WithError(b *testing.B) {
	app := fiber.New()
	app.Use(EnhancedErrorHandlingMiddleware())

	app.Get("/test", func(c *fiber.Ctx) error {
		return errors.New("test error")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		app.Test(req)
	}
}
