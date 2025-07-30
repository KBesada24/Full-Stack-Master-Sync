package middleware

import (
	"fmt"
	"strings"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
)

// ValidationConfig holds validation middleware configuration
type ValidationConfig struct {
	MaxBodySize     int64
	AllowedMethods  []string
	RequiredHeaders []string
}

// DefaultValidationConfig returns default validation configuration
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxBodySize: 10 * 1024 * 1024, // 10MB
		AllowedMethods: []string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodPatch,
			fiber.MethodDelete,
			fiber.MethodOptions,
			fiber.MethodHead,
		},
		RequiredHeaders: []string{},
	}
}

// RequestValidation creates a request validation middleware
func RequestValidation(config ...ValidationConfig) fiber.Handler {
	cfg := DefaultValidationConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Validate HTTP method
		if !isMethodAllowed(c.Method(), cfg.AllowedMethods) {
			return utils.ErrorResponse(c, fiber.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
				fmt.Sprintf("Method %s is not allowed", c.Method()), nil)
		}

		// Validate required headers
		for _, header := range cfg.RequiredHeaders {
			if c.Get(header) == "" {
				return utils.ErrorResponse(c, fiber.StatusBadRequest, "MISSING_HEADER",
					fmt.Sprintf("Required header %s is missing", header), nil)
			}
		}

		// Validate content type for POST/PUT/PATCH requests
		if isBodyMethod(c.Method()) {
			contentType := c.Get("Content-Type")
			if contentType != "" && !isValidContentType(contentType) {
				return utils.ErrorResponse(c, fiber.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE",
					"Content-Type must be application/json", nil)
			}
		}

		// Validate body size
		if len(c.Body()) > int(cfg.MaxBodySize) {
			return utils.ErrorResponse(c, fiber.StatusRequestEntityTooLarge, "BODY_TOO_LARGE",
				fmt.Sprintf("Request body exceeds maximum size of %d bytes", cfg.MaxBodySize), nil)
		}

		// Validate JSON format for requests with JSON content type
		if isBodyMethod(c.Method()) && strings.Contains(c.Get("Content-Type"), "application/json") {
			if len(c.Body()) > 0 && !utils.IsValidJSON(string(c.Body())) {
				return utils.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_JSON",
					"Request body contains invalid JSON", nil)
			}
		}

		return c.Next()
	}
}

// ValidateJSON validates JSON request body against a struct
func ValidateJSON(target interface{}) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only validate for methods that typically have a body
		if !isBodyMethod(c.Method()) {
			return c.Next()
		}

		// Skip validation if no body
		if len(c.Body()) == 0 {
			return c.Next()
		}

		// Validate JSON structure and content
		result := utils.ValidateJSON(c, target)
		if !result.IsValid {
			return utils.HandleValidationErrors(c, result)
		}

		return c.Next()
	}
}

// ValidateQuery validates query parameters
func ValidateQuery(rules map[string]string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result := utils.ValidateQuery(c, rules)
		if !result.IsValid {
			return utils.HandleValidationErrors(c, result)
		}

		return c.Next()
	}
}

// ValidateParams validates URL parameters
func ValidateParams(rules map[string]string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result := utils.ValidateParams(c, rules)
		if !result.IsValid {
			return utils.HandleValidationErrors(c, result)
		}

		return c.Next()
	}
}

// SanitizeInput sanitizes input data to prevent injection attacks
func SanitizeInput() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Sanitize query parameters
		queries := c.Queries()
		for key, value := range queries {
			sanitized, err := utils.ValidateAndSanitizeInput(value, 1000)
			if err != nil {
				return utils.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_INPUT",
					fmt.Sprintf("Invalid input in query parameter %s: %s", key, err.Error()), nil)
			}
			// Note: We can't modify the original query map, but we can validate it
			_ = sanitized // Use the sanitized value for validation
		}

		// Sanitize URL parameters
		params := c.AllParams()
		for key, value := range params {
			sanitized, err := utils.ValidateAndSanitizeInput(value, 1000)
			if err != nil {
				return utils.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_INPUT",
					fmt.Sprintf("Invalid input in URL parameter %s: %s", key, err.Error()), nil)
			}
			// Note: We can't modify URL parameters after they're parsed, but we can validate them
			_ = sanitized // Use the sanitized value for validation
		}

		return c.Next()
	}
}

// ContentTypeValidation validates content type for specific endpoints
func ContentTypeValidation(allowedTypes []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !isBodyMethod(c.Method()) {
			return c.Next()
		}

		contentType := c.Get("Content-Type")
		if contentType == "" {
			return utils.ErrorResponse(c, fiber.StatusBadRequest, "MISSING_CONTENT_TYPE",
				"Content-Type header is required", nil)
		}

		// Check if content type is allowed
		for _, allowedType := range allowedTypes {
			if strings.Contains(contentType, allowedType) {
				return c.Next()
			}
		}

		return utils.ErrorResponse(c, fiber.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE",
			fmt.Sprintf("Content-Type must be one of: %s", strings.Join(allowedTypes, ", ")), nil)
	}
}

// Helper functions

// isMethodAllowed checks if HTTP method is allowed
func isMethodAllowed(method string, allowedMethods []string) bool {
	for _, allowed := range allowedMethods {
		if method == allowed {
			return true
		}
	}
	return false
}

// isBodyMethod checks if HTTP method typically has a request body
func isBodyMethod(method string) bool {
	bodyMethods := []string{fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch}
	for _, bodyMethod := range bodyMethods {
		if method == bodyMethod {
			return true
		}
	}
	return false
}

// isValidContentType checks if content type is valid for JSON APIs
func isValidContentType(contentType string) bool {
	validTypes := []string{
		"application/json",
		"application/json; charset=utf-8",
		"text/plain", // Allow for some flexibility
	}

	contentType = strings.ToLower(strings.TrimSpace(contentType))
	for _, validType := range validTypes {
		if strings.Contains(contentType, validType) {
			return true
		}
	}
	return false
}

// RequestSizeLimit creates a middleware to limit request size
func RequestSizeLimit(maxSize int64) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if len(c.Body()) > int(maxSize) {
			return utils.ErrorResponse(c, fiber.StatusRequestEntityTooLarge, "BODY_TOO_LARGE",
				fmt.Sprintf("Request body exceeds maximum size of %d bytes", maxSize), nil)
		}
		return c.Next()
	}
}
