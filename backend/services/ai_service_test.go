package services

import (
	"context"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAIService(t *testing.T) {
	tests := []struct {
		name          string
		apiKey        string
		expectedAvail bool
	}{
		{
			name:          "with valid API key",
			apiKey:        "test-api-key",
			expectedAvail: true,
		},
		{
			name:          "without API key",
			apiKey:        "",
			expectedAvail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				OpenAIAPIKey: tt.apiKey,
			}
			logger := utils.NewLogger("debug", "json")

			service := NewAIService(cfg, nil, logger)

			assert.NotNil(t, service)
			assert.Equal(t, tt.expectedAvail, service.IsAvailable())
			assert.NotNil(t, service.rateLimiter)
			assert.NotNil(t, service.config)
		})
	}
}

func TestAIService_IsAvailable(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-api-key",
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)
	assert.True(t, service.IsAvailable())

	// Test with no API key
	cfgNoKey := &config.Config{
		OpenAIAPIKey: "",
	}

	serviceNoKey := NewAIService(cfgNoKey, nil, logger)
	assert.False(t, serviceNoKey.IsAvailable())
}

func TestAIService_GetCodeSuggestions_ServiceUnavailable(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "", // No API key to simulate unavailable service
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)
	ctx := context.Background()

	req := &models.AIRequest{
		Code:        "console.log('hello world');",
		Language:    "javascript",
		Context:     "Simple hello world example",
		RequestType: "suggestion",
		Metadata:    map[string]string{"file": "test.js"},
	}

	response, err := service.GetCodeSuggestions(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.RequestID)
	assert.Equal(t, 0.1, response.Confidence) // Low confidence for fallback
	assert.Len(t, response.Suggestions, 1)
	assert.Equal(t, "improvement", response.Suggestions[0].Type)
	assert.Contains(t, response.Analysis, "unavailable")
}

func TestAIService_GetCodeSuggestions_DifferentRequestTypes(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "", // No API key to test fallback responses
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)
	ctx := context.Background()

	testCases := []struct {
		requestType  string
		expectedType string
		expectedPrio string
	}{
		{"suggestion", "improvement", "low"},
		{"debug", "fix", "low"},
		{"optimize", "improvement", "low"},
		{"refactor", "improvement", "low"},
		{"explain", "improvement", "low"},
	}

	for _, tc := range testCases {
		t.Run(tc.requestType, func(t *testing.T) {
			req := &models.AIRequest{
				Code:        "console.log('test');",
				Language:    "javascript",
				RequestType: tc.requestType,
			}

			response, err := service.GetCodeSuggestions(ctx, req)

			require.NoError(t, err)
			assert.NotNil(t, response)
			assert.Len(t, response.Suggestions, 1)
			assert.Equal(t, tc.expectedType, response.Suggestions[0].Type)
			assert.Equal(t, tc.expectedPrio, response.Suggestions[0].Priority)
		})
	}
}

func TestAIService_AnalyzeLogs_ServiceUnavailable(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "", // No API key to simulate unavailable service
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)
	ctx := context.Background()

	req := &models.AILogAnalysisRequest{
		Logs: []models.LogEntry{
			{
				ID:        "1",
				Timestamp: time.Now(),
				Level:     "error",
				Source:    "frontend",
				Message:   "Test error message",
				Context:   map[string]interface{}{"test": "data"},
			},
		},
		AnalysisType: "error_detection",
	}

	response, err := service.AnalyzeLogs(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Summary, "unavailable")
	assert.Equal(t, 0.1, response.Confidence) // Low confidence for fallback
	assert.Len(t, response.Issues, 1)
	assert.Equal(t, "service_unavailable", response.Issues[0].Type)
	assert.NotEmpty(t, response.Suggestions)
}

func TestAIService_buildCodePrompt(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)

	tests := []struct {
		name        string
		requestType string
		expected    string
	}{
		{
			name:        "suggestion request",
			requestType: "suggestion",
			expected:    "Please analyze the following code and provide suggestions for improvement:",
		},
		{
			name:        "debug request",
			requestType: "debug",
			expected:    "Please help debug the following code and identify potential issues:",
		},
		{
			name:        "optimize request",
			requestType: "optimize",
			expected:    "Please analyze the following code and suggest optimizations:",
		},
		{
			name:        "refactor request",
			requestType: "refactor",
			expected:    "Please suggest refactoring improvements for the following code:",
		},
		{
			name:        "explain request",
			requestType: "explain",
			expected:    "Please explain what the following code does:",
		},
		{
			name:        "unknown request",
			requestType: "unknown",
			expected:    "Please analyze the following code:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &models.AIRequest{
				Code:        "console.log('test');",
				Language:    "javascript",
				Context:     "test context",
				RequestType: tt.requestType,
			}

			prompt := service.buildCodePrompt(req)

			assert.Contains(t, prompt, tt.expected)
			assert.Contains(t, prompt, "Language: javascript")
			assert.Contains(t, prompt, "console.log('test');")
			assert.Contains(t, prompt, "Context: test context")
		})
	}
}

func TestAIService_buildLogAnalysisPrompt(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)

	req := &models.AILogAnalysisRequest{
		Logs: []models.LogEntry{
			{
				ID:         "1",
				Timestamp:  time.Now(),
				Level:      "error",
				Source:     "frontend",
				Message:    "Test error message",
				StackTrace: "Error stack trace",
			},
			{
				ID:        "2",
				Timestamp: time.Now(),
				Level:     "warn",
				Source:    "backend",
				Message:   "Test warning message",
			},
		},
		AnalysisType: "error_detection",
	}

	prompt := service.buildLogAnalysisPrompt(req)

	assert.Contains(t, prompt, "error_detection")
	assert.Contains(t, prompt, "Test error message")
	assert.Contains(t, prompt, "Test warning message")
	assert.Contains(t, prompt, "Error stack trace")
	assert.Contains(t, prompt, "Level: error")
	assert.Contains(t, prompt, "Source: frontend")
	assert.Contains(t, prompt, "summary of the main issues")
}

func TestAIService_parseCodeSuggestions(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)

	req := &models.AIRequest{
		Code:        "console.log('test');",
		Language:    "javascript",
		RequestType: "suggestion",
	}

	content := "Here are some suggestions for your code..."
	suggestions := service.parseCodeSuggestions(content, req)

	assert.Len(t, suggestions, 1)
	assert.Equal(t, "improvement", suggestions[0].Type)
	assert.Equal(t, "AI-generated code suggestion", suggestions[0].Description)
	assert.Equal(t, content, suggestions[0].Code)
	assert.Equal(t, 1, suggestions[0].LineNumber)
	assert.Equal(t, "medium", suggestions[0].Priority)
	assert.NotEmpty(t, suggestions[0].Reasoning)
}

func TestAIService_parseLogAnalysis(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)

	req := &models.AILogAnalysisRequest{
		AnalysisType: "error_detection",
	}

	content := "Analysis of your logs shows several issues..."
	analysis := service.parseLogAnalysis(content, req)

	assert.Equal(t, content, analysis.Summary)
	assert.Len(t, analysis.Issues, 1)
	assert.Equal(t, "general", analysis.Issues[0].Type)
	assert.Equal(t, "medium", analysis.Issues[0].Severity)
	assert.Len(t, analysis.Patterns, 1)
	assert.NotEmpty(t, analysis.Suggestions)
}

func TestAIService_updateAvailability(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)

	// Initially available
	assert.True(t, service.IsAvailable())

	// Update to unavailable
	testErr := assert.AnError
	service.updateAvailability(false, testErr)

	assert.False(t, service.IsAvailable())

	status := service.GetStatus()
	assert.False(t, status["available"].(bool))
	assert.Equal(t, testErr.Error(), status["last_error"].(string))
	assert.NotNil(t, status["last_check"])

	// Update back to available
	service.updateAvailability(true, nil)
	assert.True(t, service.IsAvailable())

	status = service.GetStatus()
	assert.True(t, status["available"].(bool))
}

func TestAIService_GetStatus(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "test-key",
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)

	status := service.GetStatus()

	assert.Contains(t, status, "available")
	assert.Contains(t, status, "last_check")
	assert.True(t, status["available"].(bool))
	assert.IsType(t, time.Time{}, status["last_check"])

	// Test with error
	testErr := assert.AnError
	service.updateAvailability(false, testErr)

	status = service.GetStatus()
	assert.False(t, status["available"].(bool))
	assert.Equal(t, testErr.Error(), status["last_error"].(string))
}

func TestAIService_RateLimiting(t *testing.T) {
	cfg := &config.Config{
		OpenAIAPIKey: "", // No API key to avoid actual API calls
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)
	ctx := context.Background()

	req := &models.AIRequest{
		Code:        "console.log('test');",
		Language:    "javascript",
		RequestType: "suggestion",
	}

	// Make multiple requests quickly
	for i := 0; i < 5; i++ {
		response, err := service.GetCodeSuggestions(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, response)
	}

	// The rate limiter should allow these requests since we're using fallback responses
	// In a real scenario with API calls, this would test actual rate limiting
}

// Benchmark tests
func BenchmarkAIService_GetCodeSuggestions_Fallback(b *testing.B) {
	cfg := &config.Config{
		OpenAIAPIKey: "", // No API key to test fallback performance
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)
	ctx := context.Background()

	req := &models.AIRequest{
		Code:        "console.log('hello world');",
		Language:    "javascript",
		Context:     "Simple hello world example",
		RequestType: "suggestion",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetCodeSuggestions(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAIService_AnalyzeLogs_Fallback(b *testing.B) {
	cfg := &config.Config{
		OpenAIAPIKey: "", // No API key to test fallback performance
	}
	logger := utils.NewLogger("debug", "json")

	service := NewAIService(cfg, nil, logger)
	ctx := context.Background()

	req := &models.AILogAnalysisRequest{
		Logs: []models.LogEntry{
			{
				ID:        "1",
				Timestamp: time.Now(),
				Level:     "error",
				Source:    "frontend",
				Message:   "Test error message",
			},
		},
		AnalysisType: "error_detection",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.AnalyzeLogs(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
