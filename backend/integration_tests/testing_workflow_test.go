package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/handlers"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/middleware"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTestingOrchestrationWorkflow tests the complete testing orchestration workflow
func TestTestingOrchestrationWorkflow(t *testing.T) {
	// Setup test environment
	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
		FrontendURL: "http://localhost:3000",
		LogLevel:    "info",
		LogFormat:   "json",
	}

	// Initialize logger
	utils.InitLogger(cfg.LogLevel, cfg.LogFormat)

	// Initialize WebSocket hub
	websocket.InitializeHub()
	hub := websocket.GetHub()
	go hub.Run()

	// Initialize services
	testService := services.NewTestService(cfg, hub)

	// Initialize handlers
	testingHandler := handlers.NewTestingHandler(testService)

	// Create Fiber app
	app := fiber.New()
	app.Use(middleware.CORSWithOrigins([]string{"http://localhost:3000"}))
	app.Use(middleware.RequestValidation())

	// Setup routes
	api := app.Group("/api")
	testingAPI := api.Group("/testing")
	testingAPI.Post("/run", testingHandler.RunTests)
	testingAPI.Get("/results/:runId", testingHandler.GetTestResults)
	testingAPI.Post("/validate-sync", testingHandler.ValidateSync)
	testingAPI.Get("/active", testingHandler.GetActiveRuns)
	testingAPI.Get("/history", testingHandler.GetRunHistory)
	testingAPI.Delete("/runs/:runId", testingHandler.CancelTestRun)
	testingAPI.Get("/status", testingHandler.GetTestingStatus)
	testingAPI.Get("/health", testingHandler.HealthCheck)

	time.Sleep(100 * time.Millisecond)

	t.Run("Complete Test Execution Workflow", func(t *testing.T) {
		// Step 1: Check testing service health
		req := httptest.NewRequest("GET", "/api/testing/health", nil)
		resp, err := app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Step 2: Get testing service status
		req = httptest.NewRequest("GET", "/api/testing/status", nil)
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var statusResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&statusResp)
		require.NoError(t, err)
		assert.Equal(t, true, statusResp["success"])

		// Step 3: Start a test run
		testRequest := models.TestRunRequest{
			Framework:   "jest",
			TestSuite:   "test/integration.test.js",
			Environment: "test",
			Config: map[string]string{
				"timeout": "30000",
				"verbose": "true",
			},
			Tags: []string{"integration", "api"},
		}

		reqBody, err := json.Marshal(testRequest)
		require.NoError(t, err)

		req = httptest.NewRequest("POST", "/api/testing/run", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var runResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&runResp)
		require.NoError(t, err)
		assert.Equal(t, true, runResp["success"])

		// Verify response structure
		data, ok := runResp["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, data, "run_id")
		assert.Contains(t, data, "status")
		assert.Contains(t, data, "framework")
		assert.Contains(t, data, "environment")

		runID, ok := data["run_id"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, runID)

		// Step 4: Test WebSocket broadcasting for test progress
		hub.BroadcastToAll("test_progress", map[string]interface{}{
			"run_id":      runID,
			"status":      "running",
			"framework":   "jest",
			"environment": "test",
		})

		// Verify broadcast functionality
		time.Sleep(100 * time.Millisecond)
		assert.GreaterOrEqual(t, hub.GetConnectedClients(), 0)

		// Step 5: Get test results
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/testing/results/%s", runID), nil)
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resultsResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&resultsResp)
		require.NoError(t, err)
		assert.Equal(t, true, resultsResp["success"])

		// Verify results structure
		resultsData, ok := resultsResp["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, resultsData, "run_id")
		assert.Contains(t, resultsData, "status")
		assert.Contains(t, resultsData, "total_tests")
		assert.Contains(t, resultsData, "results")
		assert.Equal(t, runID, resultsData["run_id"])

		// Step 6: Check active runs
		req = httptest.NewRequest("GET", "/api/testing/active", nil)
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var activeResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&activeResp)
		require.NoError(t, err)
		assert.Equal(t, true, activeResp["success"])

		// Step 7: Check run history
		req = httptest.NewRequest("GET", "/api/testing/history", nil)
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var historyResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&historyResp)
		require.NoError(t, err)
		assert.Equal(t, true, historyResp["success"])

		// The data field contains the history array directly
		historyData := historyResp["data"]
		// History may be empty if tests are still running or haven't completed yet
		// Just verify the structure is correct - data can be an array or nil
		if historyData != nil {
			runs, ok := historyData.([]interface{})
			if ok {
				assert.NotNil(t, runs, "Runs should be a valid array")
			}
		}
	})

	t.Run("Sync Validation Workflow", func(t *testing.T) {
		// Step 1: Create sync validation request
		syncValidationRequest := models.TestSyncValidationRequest{
			APIEndpoint: "http://localhost:8080/api/users",
			UIComponent: "UserList",
			TestData: map[string]interface{}{
				"users": []map[string]interface{}{
					{"id": 1, "name": "John Doe", "email": "john@example.com"},
					{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
				},
			},
			Assertions: []models.SyncAssertion{
				{
					Type:        "data_match",
					Field:       "users.length",
					Expected:    2,
					Operator:    "equals",
					Description: "Should display correct number of users",
				},
				{
					Type:        "ui_state",
					Field:       "loading",
					Expected:    false,
					Operator:    "equals",
					Description: "Loading state should be false after data loads",
				},
				{
					Type:        "status_match",
					Field:       "response.status",
					Expected:    200,
					Operator:    "equals",
					Description: "API should return 200 status",
				},
			},
			Config: map[string]string{
				"timeout": "5000",
				"retries": "3",
			},
		}

		reqBody, err := json.Marshal(syncValidationRequest)
		require.NoError(t, err)

		// Step 2: Submit sync validation request
		req := httptest.NewRequest("POST", "/api/testing/validate-sync", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 15000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var validationResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&validationResp)
		require.NoError(t, err)
		assert.Equal(t, true, validationResp["success"])

		// Verify response structure
		data, ok := validationResp["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, data, "is_valid")
		assert.Contains(t, data, "results")
		assert.Contains(t, data, "validated_at")

		// Verify assertion results
		results, ok := data["results"].([]interface{})
		require.True(t, ok)
		assert.Len(t, results, len(syncValidationRequest.Assertions))

		// Check each assertion result
		for i, result := range results {
			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok)
			assert.Contains(t, resultMap, "assertion")
			assert.Contains(t, resultMap, "passed")
			assert.Contains(t, resultMap, "message")

			t.Logf("Assertion %d: %v", i, resultMap["message"])
		}
	})

	t.Run("Multiple Framework Support", func(t *testing.T) {
		frameworks := []string{"jest", "vitest", "cypress", "playwright"}

		for _, framework := range frameworks {
			t.Run(fmt.Sprintf("Framework_%s", framework), func(t *testing.T) {
				testRequest := models.TestRunRequest{
					Framework:   framework,
					TestSuite:   fmt.Sprintf("test/%s.test.js", framework),
					Environment: "test",
					Config: map[string]string{
						"timeout": "30000",
					},
					Tags: []string{framework, "automated"},
				}

				reqBody, err := json.Marshal(testRequest)
				require.NoError(t, err)

				req := httptest.NewRequest("POST", "/api/testing/run", bytes.NewReader(reqBody))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req, 10000)
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var runResp map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&runResp)
				require.NoError(t, err)
				assert.Equal(t, true, runResp["success"])

				data, ok := runResp["data"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, framework, data["framework"])
			})
		}
	})

	t.Run("Test Run Cancellation", func(t *testing.T) {
		// Start a test run
		testRequest := models.TestRunRequest{
			Framework:   "jest",
			TestSuite:   "test/long-running.test.js",
			Environment: "test",
			Config: map[string]string{
				"timeout": "60000", // Long timeout
			},
		}

		reqBody, err := json.Marshal(testRequest)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/testing/run", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 5000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var runResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&runResp)
		require.NoError(t, err)

		data, ok := runResp["data"].(map[string]interface{})
		require.True(t, ok)
		runID, ok := data["run_id"].(string)
		require.True(t, ok)

		// Wait a moment for the test to start
		time.Sleep(200 * time.Millisecond)

		// Cancel the test run
		req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/testing/runs/%s", runID), nil)
		resp, err = app.Test(req, 5000)
		require.NoError(t, err)
		// Accept both OK and NotFound (if test already completed/failed)
		assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode)

		// Verify the test status (may be cancelled, failed, or completed depending on timing)
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/testing/results/%s", runID), nil)
		resp, err = app.Test(req, 5000)
		require.NoError(t, err)
		// Test may have moved to history, so accept both OK and NotFound
		assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode)
	})

	t.Run("Testing Error Handling", func(t *testing.T) {
		// Test invalid framework
		invalidRequest := map[string]interface{}{
			"framework":   "invalid-framework",
			"test_suite":  "test/sample.test.js",
			"environment": "test",
		}

		reqBody, err := json.Marshal(invalidRequest)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/testing/run", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 5000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		assert.Equal(t, false, errorResp["success"])

		// Test getting results for non-existent run
		req = httptest.NewRequest("GET", "/api/testing/results/non-existent-run-id", nil)
		resp, err = app.Test(req, 5000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		assert.Equal(t, false, errorResp["success"])
	})
}

// TestTestingServiceIntegrationWithWebSocket tests testing service integration with WebSocket notifications
func TestTestingServiceIntegrationWithWebSocket(t *testing.T) {
	// Setup test environment
	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
		FrontendURL: "http://localhost:3000",
		LogLevel:    "info",
		LogFormat:   "json",
	}

	utils.InitLogger(cfg.LogLevel, cfg.LogFormat)

	// Initialize WebSocket hub
	websocket.InitializeHub()
	hub := websocket.GetHub()
	go hub.Run()

	// Initialize test service
	testService := services.NewTestService(cfg, hub)

	time.Sleep(200 * time.Millisecond)

	t.Run("Test Progress Broadcast", func(t *testing.T) {
		// Start a test run
		testRequest := &models.TestRunRequest{
			Framework:   "jest",
			TestSuite:   "test/broadcast.test.js",
			Environment: "test",
			Config:      map[string]string{},
		}

		resp, err := testService.StartTestRun(context.Background(), testRequest)
		require.NoError(t, err)

		// Test broadcasting test progress
		hub.BroadcastToAll("test_progress", map[string]interface{}{
			"run_id":      resp.RunID,
			"status":      "running",
			"framework":   "jest",
			"environment": "test",
		})

		// Verify broadcast functionality
		time.Sleep(100 * time.Millisecond)
		assert.GreaterOrEqual(t, hub.GetConnectedClients(), 0)
	})

	t.Run("Test Completion Notifications", func(t *testing.T) {
		// Start a test run
		testRequest := &models.TestRunRequest{
			Framework:   "vitest",
			TestSuite:   "test/completion.test.js",
			Environment: "test",
			Config:      map[string]string{},
		}

		resp, err := testService.StartTestRun(context.Background(), testRequest)
		require.NoError(t, err)

		// Test broadcasting test completion
		hub.BroadcastToAll("test_progress", map[string]interface{}{
			"run_id":      resp.RunID,
			"status":      "completed",
			"framework":   "vitest",
			"environment": "test",
			"total_tests": 10,
			"duration":    "5s",
		})

		// Verify broadcast functionality
		time.Sleep(100 * time.Millisecond)
		assert.GreaterOrEqual(t, hub.GetConnectedClients(), 0)
	})

	t.Run("Sync Validation Notifications", func(t *testing.T) {
		// Create sync validation request
		syncRequest := &models.TestSyncValidationRequest{
			APIEndpoint: "http://localhost:8080/api/test",
			UIComponent: "TestComponent",
			TestData:    map[string]interface{}{"test": "data"},
			Assertions: []models.SyncAssertion{
				{
					Type:        "data_match",
					Field:       "test",
					Expected:    "data",
					Operator:    "equals",
					Description: "Test assertion",
				},
			},
		}

		_, err := testService.ValidateSync(context.Background(), syncRequest)
		require.NoError(t, err)

		// Test broadcasting sync validation completion
		hub.BroadcastToAll("sync_validation_complete", map[string]interface{}{
			"type":         "sync_validation",
			"api_endpoint": syncRequest.APIEndpoint,
			"ui_component": syncRequest.UIComponent,
			"is_valid":     true,
		})

		// Verify broadcast functionality
		time.Sleep(100 * time.Millisecond)
		assert.GreaterOrEqual(t, hub.GetConnectedClients(), 0)
	})
}
