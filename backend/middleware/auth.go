package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
)

// ErrorHandlingConfig holds error handling middleware configuration
type ErrorHandlingConfig struct {
	Logger         *utils.Logger
	ShowStackTrace bool
	CustomErrorMap map[int]string
}

// DefaultErrorHandlingConfig returns default error handling configuration
func DefaultErrorHandlingConfig() ErrorHandlingConfig {
	return ErrorHandlingConfig{
		Logger:         utils.GetLogger(),
		ShowStackTrace: false, // Don't show stack traces in production
		CustomErrorMap: map[int]string{
			fiber.StatusNotFound:            "The requested resource was not found",
			fiber.StatusMethodNotAllowed:    "The HTTP method is not allowed for this resource",
			fiber.StatusRequestTimeout:      "The request timed out",
			fiber.StatusTooManyRequests:     "Too many requests, please try again later",
			fiber.StatusInternalServerError: "An internal server error occurred",
			fiber.StatusBadGateway:          "Bad gateway error",
			fiber.StatusServiceUnavailable:  "Service temporarily unavailable",
			fiber.StatusGatewayTimeout:      "Gateway timeout",
		},
	}
}

// ErrorHandler creates a centralized error handling middleware
func ErrorHandler(config ...ErrorHandlingConfig) fiber.Handler {
	cfg := DefaultErrorHandlingConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Process the request
		err := c.Next()

		if err != nil {
			return handleError(c, err, cfg)
		}

		return nil
	}
}

// PanicRecovery creates a panic recovery middleware
func PanicRecovery(config ...ErrorHandlingConfig) fiber.Handler {
	cfg := DefaultErrorHandlingConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic
				traceID := utils.GetTraceID(c)
				stackTrace := string(debug.Stack())

				context := map[string]interface{}{
					"method":      c.Method(),
					"path":        c.Path(),
					"ip":          c.IP(),
					"panic_value": fmt.Sprintf("%v", r),
					"stack_trace": stackTrace,
				}

				cfg.Logger.WithTraceID(traceID).WithSource("panic").Error(
					"Panic recovered", nil, context)

				// Create error response
				details := make(map[string]string)
				if cfg.ShowStackTrace {
					details["stack_trace"] = stackTrace
					details["panic_value"] = fmt.Sprintf("%v", r)
				}

				err = utils.ErrorResponse(c, fiber.StatusInternalServerError,
					"PANIC_RECOVERED", "An unexpected error occurred", details)
			}
		}()

		return c.Next()
	}
}

// NotFoundHandler creates a custom 404 handler
func NotFoundHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return utils.NotFoundResponse(c, "Endpoint")
	}
}

// MethodNotAllowedHandler creates a custom 405 handler
func MethodNotAllowedHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return utils.ErrorResponse(c, fiber.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED", "Method not allowed for this endpoint", nil)
	}
}

// handleError handles different types of errors
func handleError(c *fiber.Ctx, err error, cfg ErrorHandlingConfig) error {
	traceID := utils.GetTraceID(c)

	// Log the error
	context := map[string]interface{}{
		"method": c.Method(),
		"path":   c.Path(),
		"ip":     c.IP(),
	}

	cfg.Logger.WithTraceID(traceID).WithSource("error").Error(
		"Request error", err, context)

	// Handle Fiber errors
	if fiberErr, ok := err.(*fiber.Error); ok {
		return handleFiberError(c, fiberErr, cfg)
	}

	// Handle custom application errors
	if appErr, ok := err.(ApplicationError); ok {
		return handleApplicationError(c, appErr)
	}

	// Handle generic errors
	return utils.InternalServerErrorResponse(c, "An unexpected error occurred")
}

// handleFiberError handles Fiber framework errors
func handleFiberError(c *fiber.Ctx, fiberErr *fiber.Error, cfg ErrorHandlingConfig) error {
	statusCode := fiberErr.Code
	message := fiberErr.Message

	// Use custom message if available
	if customMessage, exists := cfg.CustomErrorMap[statusCode]; exists {
		message = customMessage
	}

	// Map status codes to error codes
	var errorCode string
	switch statusCode {
	case fiber.StatusBadRequest:
		errorCode = "BAD_REQUEST"
	case fiber.StatusUnauthorized:
		errorCode = "UNAUTHORIZED"
	case fiber.StatusForbidden:
		errorCode = "FORBIDDEN"
	case fiber.StatusNotFound:
		errorCode = "NOT_FOUND"
	case fiber.StatusMethodNotAllowed:
		errorCode = "METHOD_NOT_ALLOWED"
	case fiber.StatusRequestTimeout:
		errorCode = "REQUEST_TIMEOUT"
	case fiber.StatusTooManyRequests:
		errorCode = "TOO_MANY_REQUESTS"
	case fiber.StatusInternalServerError:
		errorCode = "INTERNAL_ERROR"
	case fiber.StatusBadGateway:
		errorCode = "BAD_GATEWAY"
	case fiber.StatusServiceUnavailable:
		errorCode = "SERVICE_UNAVAILABLE"
	case fiber.StatusGatewayTimeout:
		errorCode = "GATEWAY_TIMEOUT"
	default:
		errorCode = "UNKNOWN_ERROR"
	}

	return utils.ErrorResponse(c, statusCode, errorCode, message, nil)
}

// ApplicationError represents a custom application error
type ApplicationError interface {
	error
	StatusCode() int
	ErrorCode() string
	Details() map[string]string
}

// CustomError implements ApplicationError
type CustomError struct {
	Code         string
	Message      string
	Status       int
	ErrorDetails map[string]string
}

func (e CustomError) Error() string {
	return e.Message
}

func (e CustomError) StatusCode() int {
	return e.Status
}

func (e CustomError) ErrorCode() string {
	return e.Code
}

func (e CustomError) Details() map[string]string {
	return e.ErrorDetails
}

// NewCustomError creates a new custom error
func NewCustomError(code, message string, status int, details map[string]string) ApplicationError {
	return CustomError{
		Code:         code,
		Message:      message,
		Status:       status,
		ErrorDetails: details,
	}
}

// handleApplicationError handles custom application errors
func handleApplicationError(c *fiber.Ctx, appErr ApplicationError) error {
	return utils.ErrorResponse(c, appErr.StatusCode(), appErr.ErrorCode(),
		appErr.Error(), appErr.Details())
}

// Common error constructors

// NewValidationError creates a validation error
func NewValidationError(message string, details map[string]string) ApplicationError {
	return NewCustomError("VALIDATION_ERROR", message, fiber.StatusBadRequest, details)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) ApplicationError {
	return NewCustomError("NOT_FOUND", fmt.Sprintf("%s not found", resource),
		fiber.StatusNotFound, nil)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string) ApplicationError {
	if message == "" {
		message = "Unauthorized access"
	}
	return NewCustomError("UNAUTHORIZED", message, fiber.StatusUnauthorized, nil)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message string) ApplicationError {
	if message == "" {
		message = "Access forbidden"
	}
	return NewCustomError("FORBIDDEN", message, fiber.StatusForbidden, nil)
}

// NewServiceUnavailableError creates a service unavailable error
func NewServiceUnavailableError(service string) ApplicationError {
	message := "Service temporarily unavailable"
	if service != "" {
		message = fmt.Sprintf("%s service temporarily unavailable", service)
	}
	return NewCustomError("SERVICE_UNAVAILABLE", message, fiber.StatusServiceUnavailable, nil)
}

// Authentication middleware (future implementation)
// This will be implemented in a later task when authentication is needed
