package handlers

import (
	"strconv"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
)

// TestingHandler handles E2E testing API endpoints
type TestingHandler struct {
	testService *services.TestService
}

// NewTestingHandler creates a new testing handler instance
func NewTestingHandler(testService *services.TestService) *TestingHandler {
	return &TestingHandler{
		testService: testService,
	}
}

// RunTests handles POST /api/testing/run - triggers test execution
func (h *TestingHandler) RunTests(c *fiber.Ctx) error {
	var req models.TestRunRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST",
			"Invalid request body", map[string]string{
				"error": err.Error(),
			})
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR",
			"Request validation failed", map[string]string{
				"error": err.Error(),
			})
	}

	// Start test run
	response, err := h.testService.StartTestRun(c.Context(), &req)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "TEST_START_ERROR",
			"Failed to start test run", map[string]string{
				"error": err.Error(),
			})
	}

	return utils.SuccessResponse(c, "Test run started successfully", response)
}

// GetTestResults handles GET /api/testing/results/:runId - retrieves test results
func (h *TestingHandler) GetTestResults(c *fiber.Ctx) error {
	runID := c.Params("runId")
	if runID == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "MISSING_RUN_ID",
			"Run ID is required", nil)
	}

	// Get test results
	results, err := h.testService.GetTestResults(runID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "TEST_RUN_NOT_FOUND",
			"Test run not found", map[string]string{
				"run_id": runID,
				"error":  err.Error(),
			})
	}

	return utils.SuccessResponse(c, "Test results retrieved successfully", results)
}

// ValidateSync handles POST /api/testing/validate-sync - validates API-UI synchronization
func (h *TestingHandler) ValidateSync(c *fiber.Ctx) error {
	var req models.TestSyncValidationRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST",
			"Invalid request body", map[string]string{
				"error": err.Error(),
			})
	}

	// Validate request
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR",
			"Request validation failed", map[string]string{
				"error": err.Error(),
			})
	}

	// Validate sync
	response, err := h.testService.ValidateSync(c.Context(), &req)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "SYNC_VALIDATION_ERROR",
			"Failed to validate synchronization", map[string]string{
				"error": err.Error(),
			})
	}

	return utils.SuccessResponse(c, "Sync validation completed", response)
}

// GetActiveRuns handles GET /api/testing/active - gets all active test runs
func (h *TestingHandler) GetActiveRuns(c *fiber.Ctx) error {
	activeRuns := h.testService.GetActiveRuns()
	return utils.SuccessResponse(c, "Active test runs retrieved successfully", activeRuns)
}

// GetRunHistory handles GET /api/testing/history - gets test run history
func (h *TestingHandler) GetRunHistory(c *fiber.Ctx) error {
	// Parse limit parameter
	limitStr := c.Query("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	history := h.testService.GetRunHistory(limit)
	return utils.SuccessResponse(c, "Test run history retrieved successfully", history)
}

// CancelTestRun handles DELETE /api/testing/runs/:runId - cancels a test run
func (h *TestingHandler) CancelTestRun(c *fiber.Ctx) error {
	runID := c.Params("runId")
	if runID == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "MISSING_RUN_ID",
			"Run ID is required", nil)
	}

	// Cancel test run
	err := h.testService.CancelTestRun(runID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "TEST_RUN_NOT_FOUND",
			"Failed to cancel test run", map[string]string{
				"run_id": runID,
				"error":  err.Error(),
			})
	}

	return utils.SuccessResponse(c, "Test run cancelled successfully", fiber.Map{
		"run_id": runID,
		"status": "cancelled",
	})
}

// GetTestingStatus handles GET /api/testing/status - gets testing service status
func (h *TestingHandler) GetTestingStatus(c *fiber.Ctx) error {
	status := h.testService.GetStatus()
	return utils.SuccessResponse(c, "Testing service status retrieved successfully", status)
}

// HealthCheck handles GET /api/testing/health - testing service health check
func (h *TestingHandler) HealthCheck(c *fiber.Ctx) error {
	health := fiber.Map{
		"status":  "healthy",
		"service": "testing",
		"message": "Testing service is operational",
	}

	// Add service-specific health checks
	status := h.testService.GetStatus()
	health["details"] = status

	return utils.SuccessResponse(c, "Testing service health check passed", health)
}
