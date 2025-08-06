package handlers

import (
	"strings"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/websocket"
	"github.com/gofiber/fiber/v2"
)

// SyncServiceInterface defines the interface for sync service operations
type SyncServiceInterface interface {
	ConnectEnvironment(req *models.SyncConnectionRequest) (*models.SyncStatusResponse, error)
	GetSyncStatus() (*models.SyncStatusResponse, error)
	ValidateEndpoint(req *models.SyncValidationRequest) (*models.SyncValidationResponse, error)
	GetEnvironments() map[string]*models.SyncEnvironment
	RemoveEnvironment(environmentName string) error
}

// SyncHandler handles environment synchronization requests
type SyncHandler struct {
	syncService SyncServiceInterface
	logger      *utils.Logger
}

// NewSyncHandler creates a new sync handler instance
func NewSyncHandler(syncService SyncServiceInterface) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
		logger:      utils.GetLogger(),
	}
}

// ConnectEnvironment handles POST /api/sync/connect requests
func (h *SyncHandler) ConnectEnvironment(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Received sync connection request", map[string]interface{}{
		"method": c.Method(),
		"path":   c.Path(),
		"ip":     c.IP(),
	})

	// Parse request body
	var req models.SyncConnectionRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.WithTraceID(traceID).Error("Failed to parse sync connection request", err, map[string]interface{}{
			"body": string(c.Body()),
		})
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body format", map[string]string{
			"error": err.Error(),
		})
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		h.logger.WithTraceID(traceID).Error("Sync connection request validation failed", err, map[string]interface{}{
			"request": req,
		})
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", map[string]string{
			"validation_error": err.Error(),
		})
	}

	// Connect to environment
	response, err := h.syncService.ConnectEnvironment(&req)
	if err != nil {
		h.logger.WithTraceID(traceID).Error("Failed to connect to sync environment", err, map[string]interface{}{
			"environment":  req.Environment,
			"frontend_url": req.FrontendURL,
			"backend_url":  req.BackendURL,
		})
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "CONNECTION_FAILED", "Failed to connect to environment", map[string]string{
			"error": err.Error(),
		})
	}

	// Broadcast sync status update via WebSocket
	websocket.BroadcastSyncUpdate(map[string]interface{}{
		"action":      "environment_connected",
		"environment": req.Environment,
		"status":      response.Status,
		"connected":   response.Connected,
		"timestamp":   response.LastSync,
	})

	h.logger.WithTraceID(traceID).Info("Sync environment connected successfully", map[string]interface{}{
		"environment": req.Environment,
		"status":      response.Status,
		"connected":   response.Connected,
	})

	return utils.SuccessResponse(c, "Environment connected successfully", response)
}

// GetSyncStatus handles GET /api/sync/status requests
func (h *SyncHandler) GetSyncStatus(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Received sync status request", map[string]interface{}{
		"method": c.Method(),
		"path":   c.Path(),
		"ip":     c.IP(),
	})

	// Get sync status
	response, err := h.syncService.GetSyncStatus()
	if err != nil {
		h.logger.WithTraceID(traceID).Error("Failed to get sync status", err, nil)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "STATUS_RETRIEVAL_FAILED", "Failed to retrieve sync status", map[string]string{
			"error": err.Error(),
		})
	}

	h.logger.WithTraceID(traceID).Debug("Sync status retrieved successfully", map[string]interface{}{
		"status":       response.Status,
		"connected":    response.Connected,
		"environments": len(response.Environments),
	})

	return utils.SuccessResponse(c, "Sync status retrieved successfully", response)
}

// ValidateEndpoint handles POST /api/sync/validate requests
func (h *SyncHandler) ValidateEndpoint(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Received endpoint validation request", map[string]interface{}{
		"method": c.Method(),
		"path":   c.Path(),
		"ip":     c.IP(),
	})

	// Parse request body
	var req models.SyncValidationRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.WithTraceID(traceID).Error("Failed to parse validation request", err, map[string]interface{}{
			"body": string(c.Body()),
		})
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body format", map[string]string{
			"error": err.Error(),
		})
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		h.logger.WithTraceID(traceID).Error("Validation request validation failed", err, map[string]interface{}{
			"request": req,
		})
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", map[string]string{
			"validation_error": err.Error(),
		})
	}

	// Validate endpoint compatibility
	response, err := h.syncService.ValidateEndpoint(&req)
	if err != nil {
		h.logger.WithTraceID(traceID).Error("Failed to validate endpoint", err, map[string]interface{}{
			"frontend_endpoint": req.FrontendEndpoint,
			"backend_endpoint":  req.BackendEndpoint,
			"method":            req.Method,
		})
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "VALIDATION_FAILED", "Failed to validate endpoint", map[string]string{
			"error": err.Error(),
		})
	}

	// Broadcast validation result via WebSocket if there are issues
	if !response.IsCompatible {
		websocket.BroadcastSyncUpdate(map[string]interface{}{
			"action":            "validation_issues_detected",
			"frontend_endpoint": req.FrontendEndpoint,
			"backend_endpoint":  req.BackendEndpoint,
			"issues_count":      len(response.Issues),
			"is_compatible":     response.IsCompatible,
			"timestamp":         response.ValidatedAt,
		})
	}

	h.logger.WithTraceID(traceID).Info("Endpoint validation completed", map[string]interface{}{
		"frontend_endpoint": req.FrontendEndpoint,
		"backend_endpoint":  req.BackendEndpoint,
		"is_compatible":     response.IsCompatible,
		"issues_count":      len(response.Issues),
	})

	return utils.SuccessResponse(c, "Endpoint validation completed", response)
}

// GetEnvironments handles GET /api/sync/environments requests
func (h *SyncHandler) GetEnvironments(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	h.logger.WithTraceID(traceID).Info("Received environments list request", map[string]interface{}{
		"method": c.Method(),
		"path":   c.Path(),
		"ip":     c.IP(),
	})

	// Get all environments
	environments := h.syncService.GetEnvironments()

	h.logger.WithTraceID(traceID).Debug("Environments retrieved successfully", map[string]interface{}{
		"count": len(environments),
	})

	return utils.SuccessResponse(c, "Environments retrieved successfully", map[string]interface{}{
		"environments": environments,
		"count":        len(environments),
	})
}

// RemoveEnvironment handles DELETE /api/sync/environments/:name requests
func (h *SyncHandler) RemoveEnvironment(c *fiber.Ctx) error {
	traceID := utils.GetTraceID(c)
	environmentName := strings.TrimSpace(c.Params("name"))

	h.logger.WithTraceID(traceID).Info("Received environment removal request", map[string]interface{}{
		"method":      c.Method(),
		"path":        c.Path(),
		"environment": environmentName,
		"ip":          c.IP(),
	})

	if environmentName == "" {
		h.logger.WithTraceID(traceID).Error("Environment name parameter is missing", nil, map[string]interface{}{
			"path": c.Path(),
		})
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "MISSING_PARAMETER", "Environment name is required", nil)
	}

	// Remove environment
	err := h.syncService.RemoveEnvironment(environmentName)
	if err != nil {
		h.logger.WithTraceID(traceID).Error("Failed to remove environment", err, map[string]interface{}{
			"environment": environmentName,
		})
		return utils.ErrorResponse(c, fiber.StatusNotFound, "ENVIRONMENT_NOT_FOUND", "Environment not found", map[string]string{
			"environment": environmentName,
			"error":       err.Error(),
		})
	}

	// Broadcast environment removal via WebSocket
	websocket.BroadcastSyncUpdate(map[string]interface{}{
		"action":      "environment_removed",
		"environment": environmentName,
		"timestamp":   fiber.Map{"removed_at": fiber.Map{}},
	})

	h.logger.WithTraceID(traceID).Info("Environment removed successfully", map[string]interface{}{
		"environment": environmentName,
	})

	return utils.SuccessResponse(c, "Environment removed successfully", map[string]interface{}{
		"environment": environmentName,
		"removed":     true,
	})
}
