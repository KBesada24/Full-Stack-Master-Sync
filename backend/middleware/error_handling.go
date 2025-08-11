package middleware

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// EnhancedErrorHandlingConfig holds configuration for enhanced error handling middleware
type EnhancedErrorHandlingConfig struct {
	// EnableStackTrace includes stack traces in error responses (development only)
	EnableStackTrace bool
	// EnableDetailedErrors includes detailed error information
	EnableDetailedErrors bool
	// MaxStackTraceDepth limits the depth of stack traces
	MaxStackTraceDepth int
	// Logger for error logging
	Logger *utils.Logger
	// RecoveryService for error recovery
	RecoveryService *utils.ErrorRecoveryService
}

// DefaultEnhancedErrorHandlingConfig returns default configuration
func DefaultEnhancedErrorHandlingConfig() *EnhancedErrorHandlingConfig {
	return &EnhancedErrorHandlingConfig{
		EnableStackTrace:     false,
		EnableDetailedErrors: true,
		MaxStackTraceDepth:   10,
		Logger:               utils.GetLogger(),
		RecoveryService:      utils.NewErrorRecoveryService(utils.GetLogger()),
	}
}

// EnhancedErrorHandlingMiddleware creates enhanced error handling middleware
func EnhancedErrorHandlingMiddleware(config ...*EnhancedErrorHandlingConfig) fiber.Handler {
	cfg := DefaultEnhancedErrorHandlingConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Set up panic recovery
		defer func() {
			if r := recover(); r != nil {
				handleEnhancedPanic(c, r, cfg)
			}
		}()

		// Continue with the request
		err := c.Next()

		// Handle any errors that occurred
		if err != nil {
			return handleEnhancedError(c, err, cfg)
		}

		return nil
	}
}

// handleEnhancedPanic handles panic recovery
func handleEnhancedPanic(c *fiber.Ctx, panicValue interface{}, config *EnhancedErrorHandlingConfig) {
	traceID := utils.GetTraceID(c)

	// Get stack trace
	stackTrace := make([]byte, 4096)
	stackSize := runtime.Stack(stackTrace, false)
	stackTrace = stackTrace[:stackSize]

	// Log the panic
	config.Logger.WithTraceID(traceID).WithSource("error_middleware").Error(
		"Panic recovered in HTTP handler",
		fmt.Errorf("panic: %v", panicValue),
		map[string]interface{}{
			"method":      c.Method(),
			"path":        c.Path(),
			"ip":          c.IP(),
			"user_agent":  c.Get("User-Agent"),
			"panic_value": panicValue,
		},
	)

	// Use recovery service if available
	if config.RecoveryService != nil {
		config.RecoveryService.RecoverWithCallback(func(r interface{}) {
			config.Logger.WithTraceID(traceID).Info("Recovery service callback executed", map[string]interface{}{
				"panic_value": r,
			})
		})
	}

	// Create error response
	errorResponse := createEnhancedErrorResponse(
		"INTERNAL_SERVER_ERROR",
		"An unexpected error occurred",
		traceID,
		config,
		map[string]string{
			"type": "panic",
		},
	)

	// Add stack trace if enabled
	if config.EnableStackTrace {
		errorResponse.Details["stack_trace"] = string(stackTrace)
	}

	// Send error response
	c.Status(fiber.StatusInternalServerError).JSON(utils.StandardResponse{
		Success:   false,
		Message:   "Request failed due to internal error",
		Error:     errorResponse,
		Timestamp: time.Now(),
		TraceID:   traceID,
	})
}

// handleEnhancedError handles regular errors
func handleEnhancedError(c *fiber.Ctx, err error, config *EnhancedErrorHandlingConfig) error {
	traceID := utils.GetTraceID(c)

	// Determine error type and status code
	statusCode, errorCode, message := categorizeError(err)

	// Log the error
	logLevel := "error"
	if statusCode < 500 {
		logLevel = "warn"
	}

	logContext := map[string]interface{}{
		"method":      c.Method(),
		"path":        c.Path(),
		"ip":          c.IP(),
		"user_agent":  c.Get("User-Agent"),
		"status_code": statusCode,
		"error_code":  errorCode,
	}

	loggerWithContext := config.Logger.WithTraceID(traceID).WithSource("error_middleware")

	switch logLevel {
	case "warn":
		loggerWithContext.Warn(fmt.Sprintf("Request failed: %s", message), logContext)
	default:
		loggerWithContext.Error(fmt.Sprintf("Request error: %s", message), err, logContext)
	}

	// Create error details
	details := make(map[string]string)
	if config.EnableDetailedErrors {
		details["original_error"] = err.Error()
		details["error_type"] = fmt.Sprintf("%T", err)
	}

	// Add circuit breaker information if applicable
	if utils.IsCircuitBreakerError(err) {
		details["circuit_breaker"] = "open"
		details["retry_after"] = "30s"
	}

	// Add retry information if applicable
	if utils.IsRetryableError(err) {
		details["retryable"] = "true"
		details["retry_strategy"] = "exponential_backoff"
	}

	// Create error response
	errorResponse := createEnhancedErrorResponse(errorCode, message, traceID, config, details)

	// Send error response
	return c.Status(statusCode).JSON(utils.StandardResponse{
		Success:   false,
		Message:   "Request failed",
		Error:     errorResponse,
		Timestamp: time.Now(),
		TraceID:   traceID,
	})
}

// categorizeError categorizes errors and returns appropriate status code and message
func categorizeError(err error) (statusCode int, errorCode, message string) {
	if err == nil {
		return fiber.StatusOK, "SUCCESS", "OK"
	}

	errStr := err.Error()

	// Circuit breaker errors
	if utils.IsCircuitBreakerError(err) {
		return fiber.StatusServiceUnavailable, "CIRCUIT_BREAKER_OPEN", "Service temporarily unavailable due to circuit breaker"
	}

	// Retry errors
	if utils.IsRetryableError(err) {
		return fiber.StatusServiceUnavailable, "RETRY_EXHAUSTED", "Service temporarily unavailable after retry attempts"
	}

	// Fiber errors
	if fiberErr, ok := err.(*fiber.Error); ok {
		return fiberErr.Code, mapFiberErrorCode(fiberErr.Code), fiberErr.Message
	}

	// Context errors
	if err == context.Canceled {
		return fiber.StatusRequestTimeout, "REQUEST_CANCELLED", "Request was cancelled"
	}
	if err == context.DeadlineExceeded {
		return fiber.StatusRequestTimeout, "REQUEST_TIMEOUT", "Request timeout exceeded"
	}

	// Network and external service errors
	if isNetworkError(errStr) {
		return fiber.StatusBadGateway, "NETWORK_ERROR", "Network connectivity issue"
	}

	if isTimeoutError(errStr) {
		return fiber.StatusGatewayTimeout, "TIMEOUT_ERROR", "Request timeout"
	}

	if isRateLimitError(errStr) {
		return fiber.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded"
	}

	if isAuthenticationError(errStr) {
		return fiber.StatusUnauthorized, "AUTHENTICATION_FAILED", "Authentication failed"
	}

	if isAuthorizationError(errStr) {
		return fiber.StatusForbidden, "AUTHORIZATION_FAILED", "Authorization failed"
	}

	if isValidationError(errStr) {
		return fiber.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed"
	}

	if isNotFoundError(errStr) {
		return fiber.StatusNotFound, "RESOURCE_NOT_FOUND", "Requested resource not found"
	}

	// Default to internal server error
	return fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An internal error occurred"
}

// mapFiberErrorCode maps Fiber status codes to error codes
func mapFiberErrorCode(statusCode int) string {
	switch statusCode {
	case fiber.StatusBadRequest:
		return "BAD_REQUEST"
	case fiber.StatusUnauthorized:
		return "UNAUTHORIZED"
	case fiber.StatusForbidden:
		return "FORBIDDEN"
	case fiber.StatusNotFound:
		return "NOT_FOUND"
	case fiber.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case fiber.StatusRequestTimeout:
		return "REQUEST_TIMEOUT"
	case fiber.StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case fiber.StatusInternalServerError:
		return "INTERNAL_SERVER_ERROR"
	case fiber.StatusBadGateway:
		return "BAD_GATEWAY"
	case fiber.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	case fiber.StatusGatewayTimeout:
		return "GATEWAY_TIMEOUT"
	default:
		return "UNKNOWN_ERROR"
	}
}

// Error pattern detection functions
func isNetworkError(errStr string) bool {
	patterns := []string{
		"connection refused",
		"connection reset",
		"network is unreachable",
		"no route to host",
		"connection timeout",
	}
	return containsAnyPattern(errStr, patterns)
}

func isTimeoutError(errStr string) bool {
	patterns := []string{
		"timeout",
		"deadline exceeded",
		"i/o timeout",
	}
	return containsAnyPattern(errStr, patterns)
}

func isRateLimitError(errStr string) bool {
	patterns := []string{
		"rate limit",
		"too many requests",
		"quota exceeded",
	}
	return containsAnyPattern(errStr, patterns)
}

func isAuthenticationError(errStr string) bool {
	patterns := []string{
		"authentication failed",
		"invalid credentials",
		"unauthorized",
		"invalid token",
		"token expired",
	}
	return containsAnyPattern(errStr, patterns)
}

func isAuthorizationError(errStr string) bool {
	patterns := []string{
		"authorization failed",
		"access denied",
		"forbidden",
		"insufficient permissions",
	}
	return containsAnyPattern(errStr, patterns)
}

func isValidationError(errStr string) bool {
	patterns := []string{
		"validation failed",
		"invalid input",
		"bad request",
		"malformed",
		"invalid format",
	}
	return containsAnyPattern(errStr, patterns)
}

func isNotFoundError(errStr string) bool {
	patterns := []string{
		"not found",
		"does not exist",
		"no such",
	}
	return containsAnyPattern(errStr, patterns)
}

// containsAnyPattern checks if string contains any of the patterns
func containsAnyPattern(s string, patterns []string) bool {
	s = strings.ToLower(s)
	for _, pattern := range patterns {
		if strings.Contains(s, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// createEnhancedErrorResponse creates a standardized error response
func createEnhancedErrorResponse(code, message, traceID string, config *EnhancedErrorHandlingConfig, details map[string]string) *utils.ErrorInfo {
	if details == nil {
		details = make(map[string]string)
	}

	// Add correlation ID for tracking
	if traceID == "" {
		traceID = uuid.New().String()
	}

	details["correlation_id"] = traceID
	details["timestamp"] = time.Now().Format(time.RFC3339)

	return &utils.ErrorInfo{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// HealthCheckErrorHandler creates a specialized error handler for health checks
func HealthCheckErrorHandler(recoveryService *utils.ErrorRecoveryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Perform health checks
		ctx := context.WithValue(c.Context(), "trace_id", utils.GetTraceID(c))
		healthResults := recoveryService.PerformHealthChecks(ctx)

		// Check if any health checks failed
		hasFailures := false
		for _, err := range healthResults {
			if err != nil {
				hasFailures = true
				break
			}
		}

		if hasFailures {
			return utils.ServiceUnavailableResponse(c, "Health check failures detected")
		}

		return utils.SuccessResponse(c, "Health checks passed", healthResults)
	}
}
