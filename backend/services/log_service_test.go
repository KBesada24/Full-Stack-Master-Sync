package services

import (
	"context"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func TestNewLogService(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()

	service := NewLogService(mockAI, hub)

	assert.NotNil(t, service)
	assert.Equal(t, mockAI, service.aiService)
	assert.Equal(t, hub, service.wsHub)
	assert.NotNil(t, service.logs)
	assert.NotNil(t, service.alerts)
	assert.NotNil(t, service.logger)
}

func TestLogService_SubmitLogs(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	tests := []struct {
		name           string
		request        *models.LogSubmissionRequest
		expectedAccept int
		expectedReject int
		expectError    bool
	}{
		{
			name: "Valid log submission",
			request: &models.LogSubmissionRequest{
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
			expectedAccept: 1,
			expectedReject: 0,
			expectError:    false,
		},
		{
			name: "Invalid log entry - missing message",
			request: &models.LogSubmissionRequest{
				Logs: []models.LogEntry{
					{
						ID:        "test-2",
						Timestamp: time.Now(),
						Level:     "info",
						Source:    "frontend",
						Message:   "", // Invalid: empty message
					},
				},
				Source: "frontend",
			},
			expectedAccept: 0,
			expectedReject: 1,
			expectError:    false,
		},
		{
			name: "Mixed valid and invalid logs",
			request: &models.LogSubmissionRequest{
				Logs: []models.LogEntry{
					{
						ID:        "test-3",
						Timestamp: time.Now(),
						Level:     "info",
						Source:    "frontend",
						Message:   "Valid message",
					},
					{
						ID:        "test-4",
						Timestamp: time.Now(),
						Level:     "invalid", // Invalid level
						Source:    "frontend",
						Message:   "Test message",
					},
				},
				Source: "frontend",
			},
			expectedAccept: 1,
			expectedReject: 1,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := service.SubmitLogs(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.expectedAccept, response.Accepted)
				assert.Equal(t, tt.expectedReject, response.Rejected)
				assert.NotEmpty(t, response.BatchID)
				assert.NotZero(t, response.ProcessedAt)
			}
		})
	}
}

func TestLogService_AnalyzeLogs(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	// Add some test logs
	testLogs := []models.LogEntry{
		{
			ID:        "log-1",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Level:     "error",
			Source:    "frontend",
			Message:   "Database connection failed",
			Component: "auth",
		},
		{
			ID:        "log-2",
			Timestamp: time.Now().Add(-30 * time.Minute),
			Level:     "error",
			Source:    "backend",
			Message:   "Database connection failed",
			Component: "auth",
		},
		{
			ID:        "log-3",
			Timestamp: time.Now().Add(-15 * time.Minute),
			Level:     "info",
			Source:    "frontend",
			Message:   "User logged in",
			Component: "auth",
		},
	}

	// Submit test logs
	req := &models.LogSubmissionRequest{
		Logs:   testLogs,
		Source: "test",
	}
	_, err := service.SubmitLogs(context.Background(), req)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		request     *models.LogAnalysisRequest
		setupMock   func()
		expectError bool
	}{
		{
			name: "Basic log analysis without AI",
			request: &models.LogAnalysisRequest{
				Limit: 100,
			},
			setupMock: func() {
				mockAI.On("IsAvailable").Return(false)
			},
			expectError: false,
		},
		{
			name: "Log analysis with level filter",
			request: &models.LogAnalysisRequest{
				Levels: []string{"error"},
				Limit:  100,
			},
			setupMock: func() {
				mockAI.On("IsAvailable").Return(false)
			},
			expectError: false,
		},
		{
			name: "Log analysis with AI enhancement",
			request: &models.LogAnalysisRequest{
				Limit: 100,
			},
			setupMock: func() {
				mockAI.On("IsAvailable").Return(true)
				mockAI.On("AnalyzeLogs", mock.Anything, mock.Anything).Return(
					&models.AILogAnalysisResponse{
						Summary: "AI-enhanced analysis",
						Issues: []models.LogIssue{
							{
								Type:        "ai_detected",
								Count:       1,
								FirstSeen:   time.Now(),
								LastSeen:    time.Now(),
								Description: "AI detected issue",
								Severity:    "medium",
								Solution:    "AI suggested solution",
							},
						},
						Patterns: []models.LogPattern{
							{
								Pattern:     "AI pattern",
								Frequency:   1,
								Description: "AI detected pattern",
							},
						},
						Suggestions: []string{"AI suggestion"},
						AnalyzedAt:  time.Now(),
						Confidence:  0.8,
					}, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockAI.ExpectedCalls = nil
			tt.setupMock()

			ctx := context.Background()
			response, err := service.AnalyzeLogs(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.Summary)
				assert.NotNil(t, response.Issues)
				assert.NotNil(t, response.Patterns)
				assert.NotNil(t, response.Suggestions)
				assert.NotZero(t, response.AnalyzedAt)
			}

			mockAI.AssertExpectations(t)
		})
	}
}

func TestLogService_FilterLogs(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	// Add test logs with different characteristics
	now := time.Now()
	testLogs := []models.LogEntry{
		{
			ID:        "log-1",
			Timestamp: now.Add(-2 * time.Hour),
			Level:     "error",
			Source:    "frontend",
			Message:   "Error message",
			Component: "auth",
		},
		{
			ID:        "log-2",
			Timestamp: now.Add(-1 * time.Hour),
			Level:     "info",
			Source:    "backend",
			Message:   "Info message",
			Component: "api",
		},
		{
			ID:        "log-3",
			Timestamp: now.Add(-30 * time.Minute),
			Level:     "warn",
			Source:    "frontend",
			Message:   "Warning message",
			Component: "auth",
		},
	}

	service.logs = testLogs

	tests := []struct {
		name           string
		request        *models.LogAnalysisRequest
		expectedCount  int
		expectedLogIDs []string
	}{
		{
			name: "No filters - all logs",
			request: &models.LogAnalysisRequest{
				Limit: 100,
			},
			expectedCount:  3,
			expectedLogIDs: []string{"log-3", "log-2", "log-1"}, // Sorted by timestamp desc
		},
		{
			name: "Filter by level",
			request: &models.LogAnalysisRequest{
				Levels: []string{"error"},
				Limit:  100,
			},
			expectedCount:  1,
			expectedLogIDs: []string{"log-1"},
		},
		{
			name: "Filter by source",
			request: &models.LogAnalysisRequest{
				Sources: []string{"frontend"},
				Limit:   100,
			},
			expectedCount:  2,
			expectedLogIDs: []string{"log-3", "log-1"},
		},
		{
			name: "Filter by component",
			request: &models.LogAnalysisRequest{
				Components: []string{"auth"},
				Limit:      100,
			},
			expectedCount:  2,
			expectedLogIDs: []string{"log-3", "log-1"},
		},
		{
			name: "Filter by time range",
			request: &models.LogAnalysisRequest{
				TimeRange: models.TimeRange{
					Start: now.Add(-90 * time.Minute),
					End:   now,
				},
				Limit: 100,
			},
			expectedCount:  2,
			expectedLogIDs: []string{"log-3", "log-2"},
		},
		{
			name: "Search query filter",
			request: &models.LogAnalysisRequest{
				SearchQuery: "Error",
				Limit:       100,
			},
			expectedCount:  1,
			expectedLogIDs: []string{"log-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := service.filterLogs(tt.request)

			assert.Equal(t, tt.expectedCount, len(filtered))

			if len(tt.expectedLogIDs) > 0 {
				actualIDs := make([]string, len(filtered))
				for i, log := range filtered {
					actualIDs[i] = log.ID
				}
				assert.Equal(t, tt.expectedLogIDs, actualIDs)
			}
		})
	}
}

func TestLogService_DetectIssues(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	// Create logs with error patterns
	now := time.Now()
	testLogs := []models.LogEntry{
		{ID: "1", Level: "error", Message: "Database error", Timestamp: now.Add(-5 * time.Minute)},
		{ID: "2", Level: "error", Message: "Database error", Timestamp: now.Add(-4 * time.Minute)},
		{ID: "3", Level: "error", Message: "Database error", Timestamp: now.Add(-3 * time.Minute)},
		{ID: "4", Level: "error", Message: "Database error", Timestamp: now.Add(-2 * time.Minute)},
		{ID: "5", Level: "error", Message: "Database error", Timestamp: now.Add(-1 * time.Minute)},
		{ID: "6", Level: "info", Message: "Normal operation", Timestamp: now},
	}

	issues := service.detectIssues(testLogs)

	assert.Len(t, issues, 1)
	assert.Equal(t, "error_spike", issues[0].Type)
	assert.Equal(t, 5, issues[0].Count)
	assert.Contains(t, issues[0].Description, "Database error")
}

func TestLogService_DetectPatterns(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	// Create logs with recurring patterns
	now := time.Now()
	testLogs := []models.LogEntry{
		{ID: "1", Message: "User login attempt", Timestamp: now.Add(-5 * time.Minute)},
		{ID: "2", Message: "User login attempt", Timestamp: now.Add(-4 * time.Minute)},
		{ID: "3", Message: "User login attempt", Timestamp: now.Add(-3 * time.Minute)},
		{ID: "4", Message: "Different message", Timestamp: now.Add(-2 * time.Minute)},
	}

	patterns := service.detectPatterns(testLogs)

	assert.Len(t, patterns, 1)
	assert.Equal(t, "User login attempt", patterns[0].Pattern)
	assert.Equal(t, 3, patterns[0].Frequency)
}

func TestLogService_CalculateStatistics(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	// Create test logs with various characteristics
	testLogs := []models.LogEntry{
		{ID: "1", Level: "error", Source: "frontend", Component: "auth", Message: "Auth error"},
		{ID: "2", Level: "error", Source: "backend", Component: "auth", Message: "Auth error"},
		{ID: "3", Level: "info", Source: "frontend", Component: "ui", Message: "UI event"},
		{ID: "4", Level: "warn", Source: "backend", Component: "api", Message: "API warning"},
	}

	stats := service.calculateStatistics(testLogs)

	assert.Equal(t, 4, stats.TotalLogs)
	assert.Equal(t, 2, stats.LogsByLevel["error"])
	assert.Equal(t, 1, stats.LogsByLevel["info"])
	assert.Equal(t, 1, stats.LogsByLevel["warn"])
	assert.Equal(t, 2, stats.LogsBySource["frontend"])
	assert.Equal(t, 2, stats.LogsBySource["backend"])
	assert.Equal(t, 50.0, stats.ErrorRate) // 2 errors out of 4 total = 50%
	assert.Len(t, stats.TopErrors, 1)
	assert.Equal(t, "Auth error", stats.TopErrors[0].Message)
	assert.Equal(t, 2, stats.TopErrors[0].Count)
}

func TestLogService_IsCriticalLogEvent(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	tests := []struct {
		name     string
		log      *models.LogEntry
		expected bool
	}{
		{
			name: "Error level is critical",
			log: &models.LogEntry{
				Level:   "error",
				Message: "Some error",
			},
			expected: true,
		},
		{
			name: "Panic keyword is critical",
			log: &models.LogEntry{
				Level:   "info",
				Message: "Application panic occurred",
			},
			expected: true,
		},
		{
			name: "Security keyword is critical",
			log: &models.LogEntry{
				Level:   "warn",
				Message: "Security breach detected",
			},
			expected: true,
		},
		{
			name: "Normal info log is not critical",
			log: &models.LogEntry{
				Level:   "info",
				Message: "User logged in successfully",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isCriticalLogEvent(tt.log)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogService_ValidateLogEntry(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	tests := []struct {
		name        string
		log         *models.LogEntry
		expectError bool
	}{
		{
			name: "Valid log entry",
			log: &models.LogEntry{
				Message: "Test message",
				Level:   "info",
				Source:  "frontend",
			},
			expectError: false,
		},
		{
			name: "Missing message",
			log: &models.LogEntry{
				Message: "",
				Level:   "info",
				Source:  "frontend",
			},
			expectError: true,
		},
		{
			name: "Missing level",
			log: &models.LogEntry{
				Message: "Test message",
				Level:   "",
				Source:  "frontend",
			},
			expectError: true,
		},
		{
			name: "Invalid level",
			log: &models.LogEntry{
				Message: "Test message",
				Level:   "invalid",
				Source:  "frontend",
			},
			expectError: true,
		},
		{
			name: "Invalid source",
			log: &models.LogEntry{
				Message: "Test message",
				Level:   "info",
				Source:  "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateLogEntry(tt.log)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLogService_GetLogCount(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	// Initially should be 0
	assert.Equal(t, 0, service.GetLogCount())

	// Add some logs
	service.logs = []models.LogEntry{
		{ID: "1", Message: "Test 1"},
		{ID: "2", Message: "Test 2"},
	}

	assert.Equal(t, 2, service.GetLogCount())
}

func TestLogService_ClearLogs(t *testing.T) {
	mockAI := &MockAIService{}
	hub := websocket.NewHub()
	service := NewLogService(mockAI, hub)

	// Add some logs
	service.logs = []models.LogEntry{
		{ID: "1", Message: "Test 1"},
		{ID: "2", Message: "Test 2"},
	}

	assert.Equal(t, 2, service.GetLogCount())

	// Clear logs
	service.ClearLogs()

	assert.Equal(t, 0, service.GetLogCount())
	assert.Empty(t, service.logs)
}
