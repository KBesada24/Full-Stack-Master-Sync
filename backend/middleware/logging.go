package middleware

import (
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// LoggingConfig holds logging middleware configuration
type LoggingConfig struct {
	Logger          *utils.Logger
	SkipPaths       []string
	SkipSuccessLogs bool
	LogRequestBody  bool
	LogResponseBody bool
	MaxBodyLogSize  int
}

// DefaultLoggingConfig returns default logging configuration
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Logger:          utils.GetLogger(),
		SkipPaths:       []string{"/health", "/metrics"},
		SkipSuccessLogs: false,
		LogRequestBody:  false,
		LogResponseBody: false,
		MaxBodyLogSize:  1024, // 1KB
	}
}

// RequestLogging creates a request logging middleware with correlation IDs
func RequestLogging(config ...LoggingConfig) fiber.Handler {
	cfg := DefaultLoggingConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Skip logging for specified paths
		if shouldSkipPath(c.Path(), cfg.SkipPaths) {
			return c.Next()
		}

		// Generate or get trace ID
		traceID := c.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Set trace ID in context and response header
		utils.SetTraceID(c, traceID)
		c.Set("X-Trace-ID", traceID)

		// Generate request ID
		requestID := uuid.New().String()
		c.Locals("request_id", requestID)
		c.Set("X-Request-ID", requestID)

		// Record start time
		startTime := time.Now()

		// Log request
		logRequest(c, cfg, traceID, requestID)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Log response
		logResponse(c, cfg, traceID, requestID, duration, err)

		return err
	}
}

// CorrelationID creates a middleware that ensures correlation IDs are present
func CorrelationID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate or get trace ID
		traceID := c.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		// Always set the response header
		c.Set("X-Trace-ID", traceID)

		// Generate request ID
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		// Always set the response header
		c.Set("X-Request-ID", requestID)

		// Store in context
		utils.SetTraceID(c, traceID)
		c.Locals("request_id", requestID)

		return c.Next()
	}
}

// StructuredLogging creates a middleware for structured logging
func StructuredLogging(logger *utils.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		traceID := utils.GetTraceID(c)
		requestID := getRequestID(c)

		// Create logger with context
		contextLogger := logger.WithTraceID(traceID).WithSource("http").WithContext(map[string]interface{}{
			"request_id": requestID,
		})

		// Store logger in context
		c.Locals("logger", contextLogger)

		return c.Next()
	}
}

// AccessLog creates an access log middleware
func AccessLog(logger *utils.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		startTime := time.Now()

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Get correlation IDs
		traceID := utils.GetTraceID(c)
		requestID := getRequestID(c)

		// Create access log entry
		context := map[string]interface{}{
			"method":       c.Method(),
			"path":         c.Path(),
			"status_code":  c.Response().StatusCode(),
			"duration_ms":  duration.Milliseconds(),
			"ip":           c.IP(),
			"user_agent":   c.Get("User-Agent"),
			"request_id":   requestID,
			"content_type": c.Get("Content-Type"),
			"accept":       c.Get("Accept"),
		}

		// Add query parameters if present
		if len(c.Queries()) > 0 {
			context["query_params"] = c.Queries()
		}

		// Add error information if present
		if err != nil {
			context["error"] = err.Error()
		}

		// Log based on status code
		statusCode := c.Response().StatusCode()
		loggerWithContext := logger.WithTraceID(traceID).WithSource("access")

		if statusCode >= 500 {
			loggerWithContext.Error("Request completed with server error", err, context)
		} else if statusCode >= 400 {
			loggerWithContext.Warn("Request completed with client error", context)
		} else {
			loggerWithContext.Info("Request completed successfully", context)
		}

		return err
	}
}

// ErrorLogging creates an error logging middleware
func ErrorLogging(logger *utils.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		if err != nil {
			traceID := utils.GetTraceID(c)
			requestID := getRequestID(c)

			context := map[string]interface{}{
				"method":     c.Method(),
				"path":       c.Path(),
				"ip":         c.IP(),
				"request_id": requestID,
			}

			logger.WithTraceID(traceID).WithSource("error").Error(
				"Request processing error", err, context)
		}

		return err
	}
}

// Helper functions

// logRequest logs incoming request details
func logRequest(c *fiber.Ctx, cfg LoggingConfig, traceID, requestID string) {
	context := map[string]interface{}{
		"method":       c.Method(),
		"path":         c.Path(),
		"ip":           c.IP(),
		"user_agent":   c.Get("User-Agent"),
		"request_id":   requestID,
		"content_type": c.Get("Content-Type"),
	}

	// Add query parameters if present
	if len(c.Queries()) > 0 {
		context["query_params"] = c.Queries()
	}

	// Add request body if configured and not too large
	if cfg.LogRequestBody && len(c.Body()) > 0 && len(c.Body()) <= cfg.MaxBodyLogSize {
		context["request_body"] = string(c.Body())
	}

	cfg.Logger.WithTraceID(traceID).WithSource("http").Info("Request received", context)
}

// logResponse logs response details
func logResponse(c *fiber.Ctx, cfg LoggingConfig, traceID, requestID string, duration time.Duration, err error) {
	statusCode := c.Response().StatusCode()

	// Skip success logs if configured
	if cfg.SkipSuccessLogs && statusCode < 400 {
		return
	}

	context := map[string]interface{}{
		"method":      c.Method(),
		"path":        c.Path(),
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"request_id":  requestID,
	}

	// Add response body if configured and not too large
	if cfg.LogResponseBody {
		responseBody := c.Response().Body()
		if len(responseBody) > 0 && len(responseBody) <= cfg.MaxBodyLogSize {
			context["response_body"] = string(responseBody)
		}
	}

	loggerWithContext := cfg.Logger.WithTraceID(traceID).WithSource("http")

	if err != nil {
		loggerWithContext.Error("Request completed with error", err, context)
	} else if statusCode >= 500 {
		loggerWithContext.Error("Request completed with server error", nil, context)
	} else if statusCode >= 400 {
		loggerWithContext.Warn("Request completed with client error", context)
	} else {
		loggerWithContext.Info("Request completed successfully", context)
	}
}

// shouldSkipPath checks if a path should be skipped from logging
func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// getRequestID gets request ID from context
func getRequestID(c *fiber.Ctx) string {
	if requestID := c.Locals("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// GetLoggerFromContext gets the logger from fiber context
func GetLoggerFromContext(c *fiber.Ctx) *utils.LoggerWithContext {
	if logger := c.Locals("logger"); logger != nil {
		if contextLogger, ok := logger.(*utils.LoggerWithContext); ok {
			return contextLogger
		}
	}

	// Fallback to creating a new logger with context
	traceID := utils.GetTraceID(c)
	requestID := getRequestID(c)

	return utils.GetLogger().WithTraceID(traceID).WithSource("http").WithContext(map[string]interface{}{
		"request_id": requestID,
	})
}
