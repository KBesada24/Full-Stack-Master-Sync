package handlers

import (
	"context"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
)

// LogServiceInterface defines the interface for log service operations
type LogServiceInterface interface {
	SubmitLogs(ctx context.Context, req *models.LogSubmissionRequest) (*models.LogSubmissionResponse, error)
	AnalyzeLogs(ctx context.Context, req *models.LogAnalysisRequest) (*models.LogAnalysisResponse, error)
	GetLogCount() int
	ClearLogs()
}

// LoggingHandler handles logging and debugging endpoints
type LoggingHandler struct {
	logService LogServiceInterface
	logger     *utils.Logger
}

// NewLoggingHandler creates a new logging handler instance
func NewLoggingHandler(logService LogServiceInterface) *LoggingHandler {
	return &LoggingHandler{
		logService: logService,
		logger:     utils.GetLogger(),
	}
}

// SubmitLogs handles POST /api/logs/submit - accepts log entries from frontend
func (h *LoggingHandler) SubmitLogs(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Processing log submission request", nil)

	// Parse request body
	var req models.LogSubmissionRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.WithTraceID(traceID).Error("Failed to parse log submission request", err, nil)
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		h.logger.WithTraceID(traceID).Error("Log submission request validation failed", err, nil)
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", map[string]string{
			"details": err.Error(),
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Submit logs to service
	response, err := h.logService.SubmitLogs(ctx, &req)
	if err != nil {
		h.logger.WithTraceID(traceID).Error("Failed to submit logs", err, map[string]interface{}{
			"batch_id":  req.BatchID,
			"source":    req.Source,
			"log_count": len(req.Logs),
		})
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "SUBMISSION_FAILED", "Failed to submit logs", nil)
	}

	h.logger.WithTraceID(traceID).Info("Log submission completed successfully", map[string]interface{}{
		"batch_id": response.BatchID,
		"accepted": response.Accepted,
		"rejected": response.Rejected,
	})

	return utils.SuccessResponse(c, "Logs submitted successfully", response)
}

// AnalyzeLogs handles GET /api/logs/analyze - performs log analysis and pattern detection
func (h *LoggingHandler) AnalyzeLogs(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Processing log analysis request", nil)

	// Parse query parameters into analysis request
	req := &models.LogAnalysisRequest{
		Limit: 1000, // Default limit
	}

	// Parse time range
	if startTime := c.Query("start_time"); startTime != "" {
		if parsed, err := time.Parse(time.RFC3339, startTime); err == nil {
			req.TimeRange.Start = parsed
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if parsed, err := time.Parse(time.RFC3339, endTime); err == nil {
			req.TimeRange.End = parsed
		}
	}

	// Parse levels filter
	if levels := c.Query("levels"); levels != "" {
		req.Levels = utils.SplitAndTrim(levels, ",")
	}

	// Parse sources filter
	if sources := c.Query("sources"); sources != "" {
		req.Sources = utils.SplitAndTrim(sources, ",")
	}

	// Parse components filter
	if components := c.Query("components"); components != "" {
		req.Components = utils.SplitAndTrim(components, ",")
	}

	// Parse search query
	req.SearchQuery = c.Query("search")

	// Parse limit
	if limit := c.QueryInt("limit", 1000); limit > 0 && limit <= 10000 {
		req.Limit = limit
	}

	// Parse custom filters
	req.Filters = make(map[string]string)
	if userID := c.Query("user_id"); userID != "" {
		req.Filters["user_id"] = userID
	}
	if sessionID := c.Query("session_id"); sessionID != "" {
		req.Filters["session_id"] = sessionID
	}

	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		h.logger.WithTraceID(traceID).Error("Log analysis request validation failed", err, nil)
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", map[string]string{
			"details": err.Error(),
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Perform log analysis
	response, err := h.logService.AnalyzeLogs(ctx, req)
	if err != nil {
		h.logger.WithTraceID(traceID).Error("Failed to analyze logs", err, map[string]interface{}{
			"time_range": req.TimeRange,
			"filters":    req.Filters,
		})
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "ANALYSIS_FAILED", "Failed to analyze logs", nil)
	}

	h.logger.WithTraceID(traceID).Info("Log analysis completed successfully", map[string]interface{}{
		"issues_found": len(response.Issues),
		"patterns":     len(response.Patterns),
		"suggestions":  len(response.Suggestions),
	})

	return utils.SuccessResponse(c, "Log analysis completed", response)
}

// GetLogStats handles GET /api/logs/stats - returns log statistics
func (h *LoggingHandler) GetLogStats(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Processing log statistics request", nil)

	stats := map[string]interface{}{
		"total_logs": h.logService.GetLogCount(),
		"timestamp":  time.Now(),
	}

	return utils.SuccessResponse(c, "Log statistics retrieved", stats)
}

// ClearLogs handles DELETE /api/logs/clear - clears all stored logs (admin only)
func (h *LoggingHandler) ClearLogs(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Processing log clear request", nil)

	// In a production system, you would check for admin permissions here
	// For now, we'll allow it for development/testing purposes

	h.logService.ClearLogs()

	h.logger.WithTraceID(traceID).Info("All logs cleared successfully", nil)

	return utils.SuccessResponse(c, "All logs cleared successfully", map[string]interface{}{
		"cleared_at": time.Now(),
	})
}

// GetLoggingStatus handles GET /api/logs/status - returns logging service status
func (h *LoggingHandler) GetLoggingStatus(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Processing logging status request", nil)

	status := map[string]interface{}{
		"service":    "logging",
		"status":     "healthy",
		"total_logs": h.logService.GetLogCount(),
		"timestamp":  time.Now(),
		"version":    "1.0.0",
	}

	return utils.SuccessResponse(c, "Logging service status", status)
}

// HealthCheck handles GET /api/logs/health - performs logging service health check
func (h *LoggingHandler) HealthCheck(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Processing logging health check", nil)

	health := map[string]interface{}{
		"service": "logging",
		"status":  "healthy",
		"checks": map[string]interface{}{
			"log_storage": "ok",
			"service":     "ok",
		},
		"timestamp": time.Now(),
	}

	return utils.SuccessResponse(c, "Logging service health check passed", health)
}
