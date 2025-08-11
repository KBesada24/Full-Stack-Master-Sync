package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIHandler_GetCodeSuggestions(t *testing.T) {
	// Setup
	cfg := &config.Config{
		OpenAIAPIKey: "", // Empty key for testing fallback behavior
	}
	logger := utils.NewLogger("debug", "json")
	aiService := services.NewAIService(cfg, nil, logger)
	handler := NewAIHandler(aiService)

	app := fiber.New()
	app.Post("/api/ai/suggestions", handler.GetCodeSuggestions)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "Valid request with fallback response",
			requestBody: models.AIRequest{
				Code:        "function hello() { console.log('Hello World'); }",
				Language:    "javascript",
				Context:     "Simple greeting function",
				RequestType: "suggestion",
				Metadata:    map[string]string{"file": "test.js"},
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "Valid debug request",
			requestBody: models.AIRequest{
				Code:        "const x = undefined; console.log(x.length);",
				Language:    "javascript",
				Context:     "Code with potential error",
				RequestType: "debug",
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "Invalid request - missing required fields",
			requestBody: models.AIRequest{
				Code:     "function test() {}",
				Language: "javascript",
				// Missing RequestType
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Invalid request - unsupported language",
			requestBody: models.AIRequest{
				Code:        "print('hello')",
				Language:    "unsupported_language",
				RequestType: "suggestion",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Invalid request - empty code",
			requestBody: models.AIRequest{
				Code:        "",
				Language:    "python",
				RequestType: "suggestion",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "Invalid JSON body",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/ai/suggestions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			require.NoError(t, err)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response
			var response utils.StandardResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			// Assert response structure
			if tt.expectedError {
				assert.False(t, response.Success)
				assert.NotNil(t, response.Error)
			} else {
				assert.True(t, response.Success)
				assert.Nil(t, response.Error)
				assert.NotNil(t, response.Data)

				// For successful requests, verify the AI response structure
				if response.Data != nil {
					dataBytes, _ := json.Marshal(response.Data)
					var aiResponse models.AIResponse
					err = json.Unmarshal(dataBytes, &aiResponse)
					assert.NoError(t, err)
					assert.NotEmpty(t, aiResponse.RequestID)
					assert.NotEmpty(t, aiResponse.Suggestions)
				}
			}
		})
	}
}

func TestAIHandler_AnalyzeLogs(t *testing.T) {
	// Setup
	cfg := &config.Config{
		OpenAIAPIKey: "", // Empty key for testing fallback behavior
	}
	logger := utils.NewLogger("debug", "json")
	aiService := services.NewAIService(cfg, nil, logger)
	handler := NewAIHandler(aiService)

	app := fiber.New()
	app.Post("/api/ai/analyze-logs", handler.AnalyzeLogs)

	// Sample log entries for testing
	sampleLogs := []models.LogEntry{
		{
			ID:        "log1",
			Timestamp: time.Now(),
			Level:     "error",
			Source:    "frontend",
			Message:   "TypeError: Cannot read property 'length' of undefined",
			Context: map[string]interface{}{
				"file": "app.js",
				"line": 42,
			},
		},
		{
			ID:        "log2",
			Timestamp: time.Now(),
			Level:     "warn",
			Source:    "backend",
			Message:   "Database connection timeout",
			Context: map[string]interface{}{
				"database": "postgres",
				"timeout":  "30s",
			},
		},
	}

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "Valid log analysis request",
			requestBody: models.AILogAnalysisRequest{
				Logs: sampleLogs,
				TimeRange: models.TimeRange{
					Start: time.Now().Add(-1 * time.Hour),
					End:   time.Now(),
				},
				AnalysisType: "error_detection",
				Filters:      map[string]string{"level": "error"},
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "Valid pattern analysis request",
			requestBody: models.AILogAnalysisRequest{
				Logs: sampleLogs,
				TimeRange: models.TimeRange{
					Start: time.Now().Add(-2 * time.Hour),
					End:   time.Now(),
				},
				AnalysisType: "pattern_analysis",
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "Invalid request - missing logs",
			requestBody: models.AILogAnalysisRequest{
				Logs:         []models.LogEntry{},
				AnalysisType: "error_detection",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Invalid request - missing analysis type",
			requestBody: models.AILogAnalysisRequest{
				Logs: sampleLogs,
				// Missing AnalysisType
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Invalid request - unsupported analysis type",
			requestBody: models.AILogAnalysisRequest{
				Logs:         sampleLogs,
				AnalysisType: "unsupported_analysis",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "Invalid JSON body",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/ai/analyze-logs", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			require.NoError(t, err)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response
			var response utils.StandardResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			// Assert response structure
			if tt.expectedError {
				assert.False(t, response.Success)
				assert.NotNil(t, response.Error)
			} else {
				assert.True(t, response.Success)
				assert.Nil(t, response.Error)
				assert.NotNil(t, response.Data)

				// For successful requests, verify the AI log analysis response structure
				if response.Data != nil {
					dataBytes, _ := json.Marshal(response.Data)
					var aiResponse models.AILogAnalysisResponse
					err = json.Unmarshal(dataBytes, &aiResponse)
					assert.NoError(t, err)
					assert.NotEmpty(t, aiResponse.Summary)
					assert.NotNil(t, aiResponse.Issues)
					assert.NotNil(t, aiResponse.Patterns)
					assert.NotNil(t, aiResponse.Suggestions)
				}
			}
		})
	}
}

func TestAIHandler_GetAIStatus(t *testing.T) {
	// Setup
	cfg := &config.Config{
		OpenAIAPIKey: "", // Empty key for testing
	}
	logger := utils.NewLogger("debug", "json")
	aiService := services.NewAIService(cfg, nil, logger)
	handler := NewAIHandler(aiService)

	app := fiber.New()
	app.Get("/api/ai/status", handler.GetAIStatus)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/ai/status", nil)

	// Execute request
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	// Assert status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response
	var response utils.StandardResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	// Assert response structure
	assert.True(t, response.Success)
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Data)

	// Verify status data structure
	dataBytes, _ := json.Marshal(response.Data)
	var statusData map[string]interface{}
	err = json.Unmarshal(dataBytes, &statusData)
	require.NoError(t, err)

	assert.Contains(t, statusData, "service_name")
	assert.Contains(t, statusData, "service_version")
	assert.Contains(t, statusData, "status")
	assert.Contains(t, statusData, "endpoints")
	assert.Contains(t, statusData, "supported_languages")
	assert.Contains(t, statusData, "supported_request_types")
	assert.Contains(t, statusData, "supported_analysis_types")
}

func TestAIHandler_HealthCheck(t *testing.T) {
	// Setup
	cfg := &config.Config{
		OpenAIAPIKey: "", // Empty key for testing
	}
	logger := utils.NewLogger("debug", "json")
	aiService := services.NewAIService(cfg, nil, logger)
	handler := NewAIHandler(aiService)

	app := fiber.New()
	app.Get("/api/ai/health", handler.HealthCheck)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/ai/health", nil)

	// Execute request
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	// For empty API key, expect service unavailable
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	// Parse response
	var response utils.StandardResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	// Assert response structure
	assert.False(t, response.Success)
	assert.NotNil(t, response.Error)
}

func TestAIHandler_Integration(t *testing.T) {
	// This test verifies the complete integration flow
	// Setup
	cfg := &config.Config{
		OpenAIAPIKey: "", // Empty key for testing fallback behavior
	}
	logger := utils.NewLogger("debug", "json")
	aiService := services.NewAIService(cfg, nil, logger)
	handler := NewAIHandler(aiService)

	app := fiber.New()
	app.Post("/api/ai/suggestions", handler.GetCodeSuggestions)
	app.Post("/api/ai/analyze-logs", handler.AnalyzeLogs)
	app.Get("/api/ai/status", handler.GetAIStatus)
	app.Get("/api/ai/health", handler.HealthCheck)

	t.Run("Complete workflow test", func(t *testing.T) {
		// 1. Check AI status
		statusReq := httptest.NewRequest(http.MethodGet, "/api/ai/status", nil)
		statusResp, err := app.Test(statusReq, -1)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, statusResp.StatusCode)

		// 2. Get code suggestions
		suggestionReq := models.AIRequest{
			Code:        "function add(a, b) { return a + b; }",
			Language:    "javascript",
			RequestType: "optimize",
		}
		suggestionBody, _ := json.Marshal(suggestionReq)
		req := httptest.NewRequest(http.MethodPost, "/api/ai/suggestions", bytes.NewReader(suggestionBody))
		req.Header.Set("Content-Type", "application/json")

		suggestionResp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, suggestionResp.StatusCode)

		// 3. Analyze logs
		logAnalysisReq := models.AILogAnalysisRequest{
			Logs: []models.LogEntry{
				{
					ID:        "test-log",
					Timestamp: time.Now(),
					Level:     "error",
					Source:    "test",
					Message:   "Test error message",
				},
			},
			TimeRange: models.TimeRange{
				Start: time.Now().Add(-1 * time.Hour),
				End:   time.Now(),
			},
			AnalysisType: "error_detection",
		}
		logBody, _ := json.Marshal(logAnalysisReq)
		logReq := httptest.NewRequest(http.MethodPost, "/api/ai/analyze-logs", bytes.NewReader(logBody))
		logReq.Header.Set("Content-Type", "application/json")

		logResp, err := app.Test(logReq, -1)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, logResp.StatusCode)
	})
}

// Benchmark tests for performance validation
func BenchmarkAIHandler_GetCodeSuggestions(b *testing.B) {
	cfg := &config.Config{OpenAIAPIKey: ""}
	logger := utils.NewLogger("debug", "json")
	aiService := services.NewAIService(cfg, nil, logger)
	handler := NewAIHandler(aiService)

	app := fiber.New()
	app.Post("/api/ai/suggestions", handler.GetCodeSuggestions)

	requestBody := models.AIRequest{
		Code:        "function test() { return 'hello'; }",
		Language:    "javascript",
		RequestType: "suggestion",
	}
	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/ai/suggestions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkAIHandler_AnalyzeLogs(b *testing.B) {
	cfg := &config.Config{OpenAIAPIKey: ""}
	logger := utils.NewLogger("debug", "json")
	aiService := services.NewAIService(cfg, nil, logger)
	handler := NewAIHandler(aiService)

	app := fiber.New()
	app.Post("/api/ai/analyze-logs", handler.AnalyzeLogs)

	requestBody := models.AILogAnalysisRequest{
		Logs: []models.LogEntry{
			{
				ID:        "bench-log",
				Timestamp: time.Now(),
				Level:     "error",
				Source:    "benchmark",
				Message:   "Benchmark error message",
			},
		},
		AnalysisType: "error_detection",
	}
	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/ai/analyze-logs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
