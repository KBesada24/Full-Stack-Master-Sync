package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWebSocketHub implements the WebSocketHub interface for testing
type MockWebSocketHub struct {
	mock.Mock
}

func (m *MockWebSocketHub) BroadcastToAll(msgType string, data interface{}) {
	m.Called(msgType, data)
}

// TestTestingHandler_RunTests tests the RunTests endpoint
func TestTestingHandler_RunTests(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Environment: "test",
	}
	mockHub := &MockWebSocketHub{}
	testService := services.NewTestService(cfg, mockHub)
	handler := NewTestingHandler(testService)

	app := fiber.New()
	app.Post("/api/testing/run", handler.RunTests)

	tests := []struct {
		name           string
		requestBody    models.TestRunRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid test run request",
			requestBody: models.TestRunRequest{
				Framework:   "cypress",
				TestSuite:   "integration/api.spec.js",
				Environment: "development",
				Config: map[string]string{
					"baseUrl": "http://localhost:3000",
				},
			},
			expectedStatus: 200,
		},
		{
			name: "Invalid framework",
			requestBody: models.TestRunRequest{
				Framework:   "invalid",
				TestSuite:   "test.spec.js",
				Environment: "development",
			},
			expectedStatus: 400,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name: "Missing required fields",
			requestBody: models.TestRunRequest{
				Framework: "cypress",
				// Missing TestSuite and Environment
			},
			expectedStatus: 400,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			if tt.expectedStatus == 200 {
				mockHub.On("BroadcastToAll", "test_progress", mock.Anything).Return()
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/testing/run", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response body
			respBody, _ := io.ReadAll(resp.Body)
			var response map[string]interface{}
			json.Unmarshal(respBody, &response)

			if tt.expectedStatus == 200 {
				assert.Equal(t, true, response["success"])
				assert.NotNil(t, response["data"])

				// Verify response data structure
				data := response["data"].(map[string]interface{})
				assert.NotEmpty(t, data["run_id"])
				assert.Equal(t, "queued", data["status"])
				assert.Equal(t, tt.requestBody.Framework, data["framework"])
			} else {
				assert.Equal(t, false, response["success"])
				errorInfo := response["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorInfo["code"])
			}

			// Reset mock for next test
			mockHub.ExpectedCalls = nil
		})
	}
}

// TestTestingHandler_GetTestResults tests the GetTestResults endpoint
func TestTestingHandler_GetTestResults(t *testing.T) {
	// Setup
	cfg := &config.Config{Environment: "test"}
	mockHub := &MockWebSocketHub{}
	testService := services.NewTestService(cfg, mockHub)
	handler := NewTestingHandler(testService)

	app := fiber.New()
	app.Get("/api/testing/results/:runId", handler.GetTestResults)

	// Start a test run first to have results to retrieve
	mockHub.On("BroadcastToAll", "test_progress", mock.Anything).Return()

	testReq := &models.TestRunRequest{
		Framework:   "cypress",
		TestSuite:   "test.spec.js",
		Environment: "test",
	}

	response, err := testService.StartTestRun(context.Background(), testReq)
	assert.NoError(t, err)
	runID := response.RunID

	// Wait a moment for the test to start
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name           string
		runID          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid run ID",
			runID:          runID,
			expectedStatus: 200,
		},
		{
			name:           "Invalid run ID",
			runID:          "invalid-run-id",
			expectedStatus: 404,
			expectedError:  "TEST_RUN_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			url := fmt.Sprintf("/api/testing/results/%s", tt.runID)
			req := httptest.NewRequest("GET", url, nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response body
			respBody, _ := io.ReadAll(resp.Body)
			var response map[string]interface{}
			json.Unmarshal(respBody, &response)

			if tt.expectedStatus == 200 {
				assert.Equal(t, true, response["success"])
				assert.NotNil(t, response["data"])

				// Verify response data structure
				data := response["data"].(map[string]interface{})
				assert.Equal(t, tt.runID, data["run_id"])
				assert.NotEmpty(t, data["status"])
			} else {
				assert.Equal(t, false, response["success"])
				if response["error"] != nil {
					errorInfo := response["error"].(map[string]interface{})
					assert.Equal(t, tt.expectedError, errorInfo["code"])
				}
			}
		})
	}
}

// TestTestingHandler_ValidateSync tests the ValidateSync endpoint
func TestTestingHandler_ValidateSync(t *testing.T) {
	// Setup
	cfg := &config.Config{Environment: "test"}
	mockHub := &MockWebSocketHub{}
	testService := services.NewTestService(cfg, mockHub)
	handler := NewTestingHandler(testService)

	app := fiber.New()
	app.Post("/api/testing/validate-sync", handler.ValidateSync)

	tests := []struct {
		name           string
		requestBody    models.TestSyncValidationRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid sync validation request",
			requestBody: models.TestSyncValidationRequest{
				APIEndpoint: "http://localhost:8080/api/users",
				UIComponent: "UserList",
				TestData:    map[string]interface{}{"userId": 1},
				Assertions: []models.SyncAssertion{
					{
						Type:        "data_match",
						Field:       "name",
						Expected:    "John Doe",
						Operator:    "equals",
						Description: "User name should match",
					},
				},
			},
			expectedStatus: 200,
		},
		{
			name: "Invalid API endpoint",
			requestBody: models.TestSyncValidationRequest{
				APIEndpoint: "invalid-url",
				UIComponent: "UserList",
				Assertions: []models.SyncAssertion{
					{
						Type:     "data_match",
						Field:    "name",
						Expected: "John Doe",
						Operator: "equals",
					},
				},
			},
			expectedStatus: 400,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name: "Missing assertions",
			requestBody: models.TestSyncValidationRequest{
				APIEndpoint: "http://localhost:8080/api/users",
				UIComponent: "UserList",
				Assertions:  []models.SyncAssertion{}, // Empty assertions
			},
			expectedStatus: 400,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/testing/validate-sync", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response body
			respBody, _ := io.ReadAll(resp.Body)
			var response map[string]interface{}
			json.Unmarshal(respBody, &response)

			if tt.expectedStatus == 200 {
				assert.Equal(t, true, response["success"])
				assert.NotNil(t, response["data"])

				// Verify response data structure
				data := response["data"].(map[string]interface{})
				assert.Contains(t, data, "is_valid")
				assert.Contains(t, data, "results")
				assert.Contains(t, data, "validated_at")
			} else {
				assert.Equal(t, false, response["success"])
				errorInfo := response["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorInfo["code"])
			}
		})
	}
}

// TestTestingHandler_GetActiveRuns tests the GetActiveRuns endpoint
func TestTestingHandler_GetActiveRuns(t *testing.T) {
	// Setup
	cfg := &config.Config{Environment: "test"}
	mockHub := &MockWebSocketHub{}
	testService := services.NewTestService(cfg, mockHub)
	handler := NewTestingHandler(testService)

	app := fiber.New()
	app.Get("/api/testing/active", handler.GetActiveRuns)

	// Setup mock expectations
	mockHub.On("BroadcastToAll", "test_progress", mock.Anything).Return()

	// Start a test run to have active runs
	testReq := &models.TestRunRequest{
		Framework:   "cypress",
		TestSuite:   "test.spec.js",
		Environment: "test",
	}

	_, err := testService.StartTestRun(context.Background(), testReq)
	assert.NoError(t, err)

	// Create request
	req := httptest.NewRequest("GET", "/api/testing/active", nil)

	// Execute request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response body
	respBody, _ := io.ReadAll(resp.Body)
	var response map[string]interface{}
	json.Unmarshal(respBody, &response)

	assert.Equal(t, true, response["success"])
	assert.NotNil(t, response["data"])

	// Verify that we have at least one active run
	data := response["data"].(map[string]interface{})
	assert.True(t, len(data) >= 1)
}

// TestTestingHandler_GetRunHistory tests the GetRunHistory endpoint
func TestTestingHandler_GetRunHistory(t *testing.T) {
	// Setup
	cfg := &config.Config{Environment: "test"}
	mockHub := &MockWebSocketHub{}
	testService := services.NewTestService(cfg, mockHub)
	handler := NewTestingHandler(testService)

	app := fiber.New()
	app.Get("/api/testing/history", handler.GetRunHistory)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "Default limit",
			queryParams:    "",
			expectedStatus: 200,
		},
		{
			name:           "Custom limit",
			queryParams:    "?limit=5",
			expectedStatus: 200,
		},
		{
			name:           "Invalid limit",
			queryParams:    "?limit=invalid",
			expectedStatus: 200, // Should default to 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			url := "/api/testing/history" + tt.queryParams
			req := httptest.NewRequest("GET", url, nil)

			// Execute request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response body
			respBody, _ := io.ReadAll(resp.Body)
			var response map[string]interface{}
			json.Unmarshal(respBody, &response)

			assert.Equal(t, true, response["success"])
			assert.NotNil(t, response["data"])
		})
	}
}

// TestTestingHandler_GetTestingStatus tests the GetTestingStatus endpoint
func TestTestingHandler_GetTestingStatus(t *testing.T) {
	// Setup
	cfg := &config.Config{Environment: "test"}
	mockHub := &MockWebSocketHub{}
	testService := services.NewTestService(cfg, mockHub)
	handler := NewTestingHandler(testService)

	app := fiber.New()
	app.Get("/api/testing/status", handler.GetTestingStatus)

	// Create request
	req := httptest.NewRequest("GET", "/api/testing/status", nil)

	// Execute request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response body
	respBody, _ := io.ReadAll(resp.Body)
	var response map[string]interface{}
	json.Unmarshal(respBody, &response)

	assert.Equal(t, true, response["success"])
	assert.NotNil(t, response["data"])

	// Verify status data structure
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data, "active_runs")
	assert.Contains(t, data, "history_count")
	assert.Contains(t, data, "supported_frameworks")
}

// TestTestingHandler_HealthCheck tests the HealthCheck endpoint
func TestTestingHandler_HealthCheck(t *testing.T) {
	// Setup
	cfg := &config.Config{Environment: "test"}
	mockHub := &MockWebSocketHub{}
	testService := services.NewTestService(cfg, mockHub)
	handler := NewTestingHandler(testService)

	app := fiber.New()
	app.Get("/api/testing/health", handler.HealthCheck)

	// Create request
	req := httptest.NewRequest("GET", "/api/testing/health", nil)

	// Execute request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response body
	respBody, _ := io.ReadAll(resp.Body)
	var response map[string]interface{}
	json.Unmarshal(respBody, &response)

	assert.Equal(t, true, response["success"])
	assert.NotNil(t, response["data"])

	// Verify health data structure
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "healthy", data["status"])
	assert.Equal(t, "testing", data["service"])
	assert.Contains(t, data, "details")
}
