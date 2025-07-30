package utils

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// StandardResponse represents a standard API response structure
type StandardResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	TraceID   string      `json:"trace_id"`
}

// ErrorInfo represents detailed error information
type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	StandardResponse
	Pagination *PaginationInfo `json:"pagination,omitempty"`
}

// SuccessResponse creates a successful response
func SuccessResponse(c *fiber.Ctx, message string, data interface{}) error {
	response := StandardResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
		TraceID:   getTraceID(c),
	}
	return c.JSON(response)
}

// ErrorResponse creates an error response
func ErrorResponse(c *fiber.Ctx, statusCode int, code, message string, details map[string]string) error {
	response := StandardResponse{
		Success: false,
		Message: "Request failed",
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
		TraceID:   getTraceID(c),
	}
	return c.Status(statusCode).JSON(response)
}

// ValidationErrorResponse creates a validation error response
func ValidationErrorResponse(c *fiber.Ctx, errors map[string]string) error {
	return ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", errors)
}

// NotFoundResponse creates a not found error response
func NotFoundResponse(c *fiber.Ctx, resource string) error {
	return ErrorResponse(c, fiber.StatusNotFound, "NOT_FOUND", resource+" not found", nil)
}

// UnauthorizedResponse creates an unauthorized error response
func UnauthorizedResponse(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "Unauthorized access"
	}
	return ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

// ForbiddenResponse creates a forbidden error response
func ForbiddenResponse(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "Access forbidden"
	}
	return ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", message, nil)
}

// InternalServerErrorResponse creates an internal server error response
func InternalServerErrorResponse(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "Internal server error"
	}
	return ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", message, nil)
}

// BadRequestResponse creates a bad request error response
func BadRequestResponse(c *fiber.Ctx, message string, details map[string]string) error {
	if message == "" {
		message = "Bad request"
	}
	return ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", message, details)
}

// ServiceUnavailableResponse creates a service unavailable error response
func ServiceUnavailableResponse(c *fiber.Ctx, service string) error {
	message := "Service temporarily unavailable"
	if service != "" {
		message = service + " service temporarily unavailable"
	}
	return ErrorResponse(c, fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message, nil)
}

// PaginatedSuccessResponse creates a successful paginated response
func PaginatedSuccessResponse(c *fiber.Ctx, message string, data interface{}, pagination *PaginationInfo) error {
	response := PaginatedResponse{
		StandardResponse: StandardResponse{
			Success:   true,
			Message:   message,
			Data:      data,
			Timestamp: time.Now(),
			TraceID:   getTraceID(c),
		},
		Pagination: pagination,
	}
	return c.JSON(response)
}

// CreatePagination creates pagination info
func CreatePagination(page, limit, total int) *PaginationInfo {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	totalPages := (total + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	return &PaginationInfo{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}
}

// getTraceID gets or generates a trace ID for request tracking
func getTraceID(c *fiber.Ctx) string {
	// Try to get trace ID from headers first
	if traceID := c.Get("X-Trace-ID"); traceID != "" {
		return traceID
	}

	// Try to get from context locals
	if traceID := c.Locals("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}

	// Generate new trace ID
	return uuid.New().String()
}

// SetTraceID sets a trace ID in the context
func SetTraceID(c *fiber.Ctx, traceID string) {
	c.Locals("trace_id", traceID)
}

// GetTraceID gets the trace ID from context
func GetTraceID(c *fiber.Ctx) string {
	return getTraceID(c)
}
