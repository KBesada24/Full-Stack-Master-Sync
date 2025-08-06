package handlers

import (
	"context"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// AIHandler handles AI assistance API endpoints
type AIHandler struct {
	aiService *services.AIService
	validator *validator.Validate
}

// NewAIHandler creates a new AI handler instance
func NewAIHandler(aiService *services.AIService) *AIHandler {
	return &AIHandler{
		aiService: aiService,
		validator: validator.New(),
	}
}

// GetCodeSuggestions handles POST /api/ai/suggestions
func (h *AIHandler) GetCodeSuggestions(c *fiber.Ctx) error {
	// Parse request body
	var req models.AIRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request body", map[string]string{
			"error": err.Error(),
		})
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		validationErrors := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors[err.Field()] = getValidationErrorMessage(err)
		}
		return utils.ValidationErrorResponse(c, validationErrors)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get AI suggestions
	response, err := h.aiService.GetCodeSuggestions(ctx, &req)
	if err != nil {
		// Check if it's a rate limit error
		if err.Error() == "rate limit exceeded" {
			return utils.ErrorResponse(c, fiber.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
				"Too many requests. Please try again later.", nil)
		}

		// Check if AI service is unavailable
		if !h.aiService.IsAvailable() {
			return utils.ServiceUnavailableResponse(c, "AI")
		}

		// Generic error
		return utils.InternalServerErrorResponse(c, "Failed to generate code suggestions")
	}

	return utils.SuccessResponse(c, "Code suggestions generated successfully", response)
}

// AnalyzeLogs handles POST /api/ai/analyze-logs
func (h *AIHandler) AnalyzeLogs(c *fiber.Ctx) error {
	// Parse request body
	var req models.AILogAnalysisRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request body", map[string]string{
			"error": err.Error(),
		})
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		validationErrors := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors[err.Field()] = getValidationErrorMessage(err)
		}
		return utils.ValidationErrorResponse(c, validationErrors)
	}

	// Additional validation for logs
	if len(req.Logs) == 0 {
		return utils.BadRequestResponse(c, "No logs provided for analysis", map[string]string{
			"logs": "At least one log entry is required",
		})
	}

	// Limit the number of logs to prevent excessive API usage
	maxLogs := 50
	if len(req.Logs) > maxLogs {
		req.Logs = req.Logs[:maxLogs]
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Analyze logs
	response, err := h.aiService.AnalyzeLogs(ctx, &req)
	if err != nil {
		// Check if it's a rate limit error
		if err.Error() == "rate limit exceeded" {
			return utils.ErrorResponse(c, fiber.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
				"Too many requests. Please try again later.", nil)
		}

		// Check if AI service is unavailable
		if !h.aiService.IsAvailable() {
			return utils.ServiceUnavailableResponse(c, "AI")
		}

		// Generic error
		return utils.InternalServerErrorResponse(c, "Failed to analyze logs")
	}

	return utils.SuccessResponse(c, "Log analysis completed successfully", response)
}

// GetAIStatus handles GET /api/ai/status
func (h *AIHandler) GetAIStatus(c *fiber.Ctx) error {
	status := h.aiService.GetStatus()

	// Add additional status information
	statusInfo := map[string]interface{}{
		"service_name":    "AI Assistance Service",
		"service_version": "1.0.0",
		"status":          status,
		"endpoints": []string{
			"POST /api/ai/suggestions - Get code suggestions",
			"POST /api/ai/analyze-logs - Analyze logs",
			"GET /api/ai/status - Get AI service status",
		},
		"supported_languages": []string{
			"javascript", "typescript", "python", "go",
			"java", "rust", "php", "swift", "kotlin", "dart",
		},
		"supported_request_types": []string{
			"suggestion", "debug", "optimize", "refactor", "explain",
		},
		"supported_analysis_types": []string{
			"error_detection", "pattern_analysis", "performance_issues", "security_scan",
		},
	}

	message := "AI service status retrieved successfully"
	if !h.aiService.IsAvailable() {
		message = "AI service is currently unavailable"
	}

	return utils.SuccessResponse(c, message, statusInfo)
}

// HealthCheck handles GET /api/ai/health
func (h *AIHandler) HealthCheck(c *fiber.Ctx) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform health check
	err := h.aiService.HealthCheck(ctx)
	if err != nil {
		return utils.ServiceUnavailableResponse(c, "AI service health check failed")
	}

	healthInfo := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"message":   "AI service is operational",
	}

	return utils.SuccessResponse(c, "AI service health check passed", healthInfo)
}

// getValidationErrorMessage returns a user-friendly validation error message
func getValidationErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "min":
		return "Value is too short or too small"
	case "max":
		return "Value is too long or too large"
	case "oneof":
		return "Value must be one of the allowed options"
	case "url":
		return "Must be a valid URL"
	case "email":
		return "Must be a valid email address"
	default:
		return "Invalid value"
	}
}
