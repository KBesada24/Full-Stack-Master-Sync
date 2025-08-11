package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Remove duplicate interface declaration - it's already in logging.go

// MockLogService is a mock implementation of LogServiceInterface for testing
type MockLogService struct {
	mock.Mock
}

// MockAIService is a mock implementation of AIServiceInterface for testing
type MockAIService struct {
	mock.Mock
}

func (m *MockAIService) IsAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockAIService) AnalyzeLogs(ctx context.Context, req *models.AILogAnalysisRequest) (*models.AILogAnalysisResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.AILogAnalysisResponse), args.Error(1)
}

func (m *MockLogService) SubmitLogs(ctx context.Context, req *models.LogSubmissionRequest) (*models.LogSubmissionResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.LogSubmissionResponse), args.Error(1)
}

func (m *MockLogService) AnalyzeLogs(ctx context.Context, req *models.LogAnalysisRequest) (*models.LogAnalysisResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.LogAnalysisResponse), args.Error(1)
}

func (m *MockLogService) GetLogCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockLogService) ClearLogs() {
	m.Called()
}

func setupLoggingTestApp() (*fiber.App, *MockLogService) {
	app := fiber.New()
	mockService := &MockLogService{}
	handler := NewLoggingHandler(mockService)

	// Setup routes
	api := app.Group("/api")
	logs := api.Group("/logs")
	logs.Post("/submit", handler.SubmitLogs)
	logs.Get("/analyze", handler.AnalyzeLogs)
	logs.Get("/stats", handler.GetLogStats)
	logs.Delete("/clear", handler.ClearLogs)
	logs.Get("/status", handler.GetLoggingStatus)
	logs.Get("/health", handler.HealthCheck)

	return app, mockService
}

func TestLoggingHandler_SubmitLogs(t *testing.T) {
	app, mockService := setupLoggingTestApp()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func()
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name: "Valid log submission",
			requestBody: models.LogSubmissionRequest{
				Logs: []models.LogEntry{
					{
						ID:        "test-1",
						Timestamp: time.Now(),
						Level:     "info",
						Source:    "frontend",
						Message:   "Test message",
					},
				},
				Source: "frontend",
			},
			setupMock: func() {
				mockService.On("SubmitLogs", mock.Anything, mock.Anything).Return(
					&models.LogSubmissionResponse{
						Accepted:    1,
						Rejected:    0,
						BatchID:     "batch-123",
						ProcessedAt: time.Now(),
						Errors:      []string{},
					}, nil)
			},
			expectedStatus: 200,
			expectSuccess:  true,
		},
		{
			name:           "Invalid JSON body",
			requestBody:    "invalid json",
			setupMock:      func() {},
			expectedStatus: 400,
			expectSuccess:  false,
		},
		{
			name: "Missing required fields",
			requestBody: models.LogSubmissionRequest{
				Logs:   []models.LogEntry{}, // Empty logs array
				Source: "",                  // Missing source
			},
			setupMock:      func() {},
			expectedStatus: 400,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockService.ExpectedCalls = nil
			tt.setupMock()

			// Create request body
			var bodyBytes []byte
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			// Create request
			req := httptest.NewRequest("POST", "/api/logs/submit", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.Equal(t, true, response["success"])
				assert.NotNil(t, response["data"])
			} else {
				assert.Equal(t, false, response["success"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestLoggingHandler_AnalyzeLogs(t *testing.T) {
	app, mockService := setupLoggingTestApp()

	tests := []struct {
		name           string
		queryParams    string
		setupMock      func()
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:        "Basic log analysis",
			queryParams: "",
			setupMock: func() {
				mockService.On("AnalyzeLogs", mock.Anything, mock.Anything).Return(
					&models.LogAnalysisResponse{
						Summary:     "Analysis complete",
						Issues:      []models.LogIssue{},
						Patterns:    []models.LogPattern{},
						Suggestions: []string{"Review logs regularly"},
						Statistics: models.LogStatistics{
							TotalLogs:     10,
							LogsByLevel:   map[string]int{"info": 8, "error": 2},
							LogsBySource:  map[string]int{"frontend": 5, "backend": 5},
							LogsByHour:    map[string]int{},
							ErrorRate:     20.0,
							TopErrors:     []models.LogErrorSummary{},
							TopComponents: []models.LogComponentSummary{},
						},
						AnalyzedAt: time.Now(),
					}, nil)
			},
			expectedStatus: 200,
			expectSuccess:  true,
		},
		{
			name:        "Log analysis with filters",
			queryParams: "?levels=error,warn&sources=frontend&limit=50",
			setupMock: func() {
				mockService.On("AnalyzeLogs", mock.Anything, mock.MatchedBy(func(req *models.LogAnalysisRequest) bool {
					return len(req.Levels) == 2 &&
						req.Levels[0] == "error" &&
						req.Levels[1] == "warn" &&
						len(req.Sources) == 1 &&
						req.Sources[0] == "frontend" &&
						req.Limit == 50
				})).Return(
					&models.LogAnalysisResponse{
						Summary:     "Filtered analysis complete",
						Issues:      []models.LogIssue{},
						Patterns:    []models.LogPattern{},
						Suggestions: []string{},
						Statistics: models.LogStatistics{
							TotalLogs:     5,
							LogsByLevel:   map[string]int{"error": 3, "warn": 2},
							LogsBySource:  map[string]int{"frontend": 5},
							LogsByHour:    map[string]int{},
							ErrorRate:     60.0,
							TopErrors:     []models.LogErrorSummary{},
							TopComponents: []models.LogComponentSummary{},
						},
						AnalyzedAt: time.Now(),
					}, nil)
			},
			expectedStatus: 200,
			expectSuccess:  true,
		},
		{
			name:        "Log analysis with time range",
			queryParams: "?start_time=2023-01-01T00:00:00Z&end_time=2023-12-31T23:59:59Z",
			setupMock: func() {
				mockService.On("AnalyzeLogs", mock.Anything, mock.MatchedBy(func(req *models.LogAnalysisRequest) bool {
					return !req.TimeRange.Start.IsZero() && !req.TimeRange.End.IsZero()
				})).Return(
					&models.LogAnalysisResponse{
						Summary:     "Time-filtered analysis complete",
						Issues:      []models.LogIssue{},
						Patterns:    []models.LogPattern{},
						Suggestions: []string{},
						Statistics: models.LogStatistics{
							TotalLogs:     3,
							LogsByLevel:   map[string]int{"info": 3},
							LogsBySource:  map[string]int{"backend": 3},
							LogsByHour:    map[string]int{},
							ErrorRate:     0.0,
							TopErrors:     []models.LogErrorSummary{},
							TopComponents: []models.LogComponentSummary{},
						},
						AnalyzedAt: time.Now(),
					}, nil)
			},
			expectedStatus: 200,
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockService.ExpectedCalls = nil
			tt.setupMock()

			// Create request
			req := httptest.NewRequest("GET", "/api/logs/analyze"+tt.queryParams, nil)

			// Execute request
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.Equal(t, true, response["success"])
				assert.NotNil(t, response["data"])

				// Verify response structure
				data := response["data"].(map[string]interface{})
				assert.Contains(t, data, "summary")
				assert.Contains(t, data, "issues")
				assert.Contains(t, data, "patterns")
				assert.Contains(t, data, "suggestions")
				assert.Contains(t, data, "statistics")
				assert.Contains(t, data, "analyzed_at")
			} else {
				assert.Equal(t, false, response["success"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestLoggingHandler_GetLogStats(t *testing.T) {
	app, mockService := setupLoggingTestApp()

	mockService.On("GetLogCount").Return(42)

	req := httptest.NewRequest("GET", "/api/logs/stats", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(42), data["total_logs"])
	assert.Contains(t, data, "timestamp")

	mockService.AssertExpectations(t)
}

func TestLoggingHandler_ClearLogs(t *testing.T) {
	app, mockService := setupLoggingTestApp()

	mockService.On("ClearLogs").Return()

	req := httptest.NewRequest("DELETE", "/api/logs/clear", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data, "cleared_at")

	mockService.AssertExpectations(t)
}

func TestLoggingHandler_GetLoggingStatus(t *testing.T) {
	app, mockService := setupLoggingTestApp()

	mockService.On("GetLogCount").Return(100)

	req := httptest.NewRequest("GET", "/api/logs/status", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "logging", data["service"])
	assert.Equal(t, "healthy", data["status"])
	assert.Equal(t, float64(100), data["total_logs"])
	assert.Contains(t, data, "timestamp")
	assert.Contains(t, data, "version")

	mockService.AssertExpectations(t)
}

func TestLoggingHandler_HealthCheck(t *testing.T) {
	app, _ := setupLoggingTestApp()

	req := httptest.NewRequest("GET", "/api/logs/health", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "logging", data["service"])
	assert.Equal(t, "healthy", data["status"])
	assert.Contains(t, data, "checks")
	assert.Contains(t, data, "timestamp")

	checks := data["checks"].(map[string]interface{})
	assert.Equal(t, "ok", checks["log_storage"])
	assert.Equal(t, "ok", checks["service"])
}

func TestNewLoggingHandler(t *testing.T) {
	// Create a real log service for this test
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	logService := services.NewLogService(mockAI, hub)

	handler := NewLoggingHandler(logService)

	assert.NotNil(t, handler)
	assert.Equal(t, logService, handler.logService)
	assert.NotNil(t, handler.logger)
}

// Integration test with real log service
func TestLoggingHandler_Integration(t *testing.T) {
	// Create real services
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	logService := services.NewLogService(mockAI, hub)
	handler := NewLoggingHandler(logService)

	// Setup app
	app := fiber.New()
	api := app.Group("/api")
	logs := api.Group("/logs")
	logs.Post("/submit", handler.SubmitLogs)
	logs.Get("/analyze", handler.AnalyzeLogs)

	// Test log submission and analysis flow
	t.Run("Submit and analyze logs", func(t *testing.T) {
		// Setup AI mock
		mockAI.On("IsAvailable").Return(false)

		// Submit logs
		submitReq := models.LogSubmissionRequest{
			Logs: []models.LogEntry{
				{
					ID:        "test-1",
					Timestamp: time.Now(),
					Level:     "error",
					Source:    "frontend",
					Message:   "Test error message",
					Component: "auth",
				},
				{
					ID:        "test-2",
					Timestamp: time.Now(),
					Level:     "info",
					Source:    "backend",
					Message:   "Test info message",
					Component: "api",
				},
			},
			Source: "frontend",
		}

		bodyBytes, _ := json.Marshal(submitReq)
		req := httptest.NewRequest("POST", "/api/logs/submit", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Analyze logs
		req = httptest.NewRequest("GET", "/api/logs/analyze?levels=error", nil)
		resp, err = app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Equal(t, true, response["success"])
		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "summary")
		assert.Contains(t, data, "statistics")

		mockAI.AssertExpectations(t)
	})
}
