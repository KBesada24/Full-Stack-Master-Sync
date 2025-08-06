package services

import (
	"context"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestService(t *testing.T) {
	cfg := &config.Config{
		CypressBaseURL:    "http://localhost:3000",
		PlaywrightBaseURL: "http://localhost:3000",
	}

	wsHub := &models.WSHub{
		Clients:    make(map[*models.WSClient]bool),
		Broadcast:  make(chan models.WSMessage, 256),
		Register:   make(chan *models.WSClient),
		Unregister: make(chan *models.WSClient),
	}

	service := NewTestService(cfg, wsHub)

	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
	assert.Equal(t, wsHub, service.wsHub)
	assert.NotNil(t, service.activeRuns)
	assert.NotNil(t, service.runHistory)
	assert.Equal(t, 100, service.maxHistory)
}

func TestTestService_IsFrameworkSupported(t *testing.T) {
	service := createTestService()

	tests := []struct {
		framework string
		expected  bool
	}{
		{"cypress", true},
		{"playwright", true},
		{"jest", true},
		{"vitest", true},
		{"CYPRESS", true}, // Case insensitive
		{"Playwright", true},
		{"mocha", false},
		{"jasmine", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.framework, func(t *testing.T) {
			result := service.isFrameworkSupported(tt.framework)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTestService_GetEstimatedDuration(t *testing.T) {
	service := createTestService()

	tests := []struct {
		framework string
		expected  time.Duration
	}{
		{"cypress", time.Minute * 5},
		{"playwright", time.Minute * 3},
		{"jest", time.Minute * 2},
		{"vitest", time.Minute * 1},
		{"unknown", time.Minute * 5},
	}

	for _, tt := range tests {
		t.Run(tt.framework, func(t *testing.T) {
			result := service.getEstimatedDuration(tt.framework)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTestService_StartTestRun(t *testing.T) {
	service := createTestService()
	ctx := context.Background()

	req := &models.TestRunRequest{
		Framework:   "cypress",
		TestSuite:   "integration/api.spec.js",
		Environment: "development",
		Config: map[string]string{
			"baseUrl": "http://localhost:3000",
		},
	}

	response, err := service.StartTestRun(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, response.RunID)
	assert.Equal(t, "queued", response.Status)
	assert.Equal(t, "cypress", response.Framework)
	assert.Equal(t, "development", response.Environment)
	assert.Equal(t, time.Minute*5, response.EstimatedDuration)
	assert.False(t, response.StartTime.IsZero())

	// Verify the run is stored in active runs
	service.mu.RLock()
	run, exists := service.activeRuns[response.RunID]
	service.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, response.RunID, run.ID)
	assert.Equal(t, req, run.Request)
	// Status might be "queued" or "running" depending on goroutine execution timing
	assert.Contains(t, []string{"queued", "running"}, run.Status)
}

func TestTestService_StartTestRun_UnsupportedFramework(t *testing.T) {
	service := createTestService()
	ctx := context.Background()

	req := &models.TestRunRequest{
		Framework:   "mocha",
		TestSuite:   "test.spec.js",
		Environment: "development",
	}

	response, err := service.StartTestRun(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "unsupported test framework: mocha")
}

func TestTestService_GetTestResults(t *testing.T) {
	service := createTestService()

	// Test with non-existent run ID
	results, err := service.GetTestResults("non-existent")
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "test run not found")

	// Create a test run
	ctx := context.Background()
	req := &models.TestRunRequest{
		Framework:   "jest",
		TestSuite:   "unit/test.spec.js",
		Environment: "test",
	}

	response, err := service.StartTestRun(ctx, req)
	require.NoError(t, err)

	// Get results for active run
	results, err = service.GetTestResults(response.RunID)
	require.NoError(t, err)
	assert.Equal(t, response.RunID, results.RunID)
	assert.Equal(t, "queued", results.Status)

	// Add to history and test history retrieval
	service.mu.Lock()
	run := service.activeRuns[response.RunID]
	run.Results.Status = "completed"
	service.runHistory = append(service.runHistory, *run.Results)
	delete(service.activeRuns, response.RunID)
	service.mu.Unlock()

	// Get results from history
	results, err = service.GetTestResults(response.RunID)
	require.NoError(t, err)
	assert.Equal(t, response.RunID, results.RunID)
	assert.Equal(t, "completed", results.Status)
}

func TestTestService_CancelTestRun(t *testing.T) {
	service := createTestService()

	// Test cancelling non-existent run
	err := service.CancelTestRun("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test run not found or already completed")

	// Create a test run
	ctx := context.Background()
	req := &models.TestRunRequest{
		Framework:   "playwright",
		TestSuite:   "e2e/test.spec.js",
		Environment: "staging",
	}

	response, err := service.StartTestRun(ctx, req)
	require.NoError(t, err)

	// Cancel the run
	err = service.CancelTestRun(response.RunID)
	require.NoError(t, err)

	// Verify the run is cancelled
	service.mu.RLock()
	run, exists := service.activeRuns[response.RunID]
	service.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "cancelled", run.Status)
	assert.Equal(t, "cancelled", run.Results.Status)
	assert.False(t, run.EndTime.IsZero())
}

func TestTestService_ValidateSync(t *testing.T) {
	service := createTestService()
	ctx := context.Background()

	req := &models.TestSyncValidationRequest{
		APIEndpoint: "http://localhost:8080/api/test",
		UIComponent: "TestComponent",
		TestData:    map[string]interface{}{"key": "value"},
		Assertions: []models.SyncAssertion{
			{
				Type:        "data_match",
				Field:       "response.data",
				Expected:    "expected_value",
				Operator:    "equals",
				Description: "Data should match between API and UI",
			},
			{
				Type:        "status_match",
				Field:       "response.status",
				Expected:    200,
				Operator:    "equals",
				Description: "Status code should be 200",
			},
		},
	}

	response, err := service.ValidateSync(ctx, req)

	require.NoError(t, err)
	assert.True(t, response.IsValid)
	assert.Len(t, response.Results, 2)
	assert.False(t, response.ValidatedAt.IsZero())

	// Check assertion results
	for _, result := range response.Results {
		assert.True(t, result.Passed)
		assert.NotEmpty(t, result.Message)
	}
}

func TestTestService_GetActiveRuns(t *testing.T) {
	service := createTestService()
	ctx := context.Background()

	// Initially no active runs
	active := service.GetActiveRuns()
	assert.Empty(t, active)

	// Start a test run
	req := &models.TestRunRequest{
		Framework:   "vitest",
		TestSuite:   "unit/test.spec.js",
		Environment: "development",
	}

	response, err := service.StartTestRun(ctx, req)
	require.NoError(t, err)

	// Check active runs
	active = service.GetActiveRuns()
	assert.Len(t, active, 1)
	assert.Contains(t, active, response.RunID)
	assert.Equal(t, "queued", active[response.RunID].Status)
	assert.Equal(t, "vitest", active[response.RunID].Framework)
}

func TestTestService_GetRunHistory(t *testing.T) {
	service := createTestService()

	// Initially no history
	history := service.GetRunHistory(10)
	assert.Empty(t, history)

	// Add some test results to history
	testResults := []models.TestResults{
		{
			RunID:       "run-1",
			Status:      "completed",
			TotalTests:  5,
			PassedTests: 5,
			FailedTests: 0,
		},
		{
			RunID:       "run-2",
			Status:      "failed",
			TotalTests:  3,
			PassedTests: 2,
			FailedTests: 1,
		},
		{
			RunID:       "run-3",
			Status:      "completed",
			TotalTests:  10,
			PassedTests: 8,
			FailedTests: 2,
		},
	}

	service.mu.Lock()
	service.runHistory = testResults
	service.mu.Unlock()

	// Get all history
	history = service.GetRunHistory(0)
	assert.Len(t, history, 3)

	// Get limited history
	history = service.GetRunHistory(2)
	assert.Len(t, history, 2)
	assert.Equal(t, "run-2", history[0].RunID)
	assert.Equal(t, "run-3", history[1].RunID)

	// Get more than available
	history = service.GetRunHistory(10)
	assert.Len(t, history, 3)
}

func TestTestService_GetSeverityFromAssertion(t *testing.T) {
	service := createTestService()

	tests := []struct {
		assertionType string
		expected      string
	}{
		{"data_match", "critical"},
		{"status_match", "warning"},
		{"timing_match", "info"},
		{"ui_state", "warning"},
		{"unknown", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.assertionType, func(t *testing.T) {
			assertion := models.SyncAssertion{Type: tt.assertionType}
			result := service.getSeverityFromAssertion(assertion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTestService_GetSuggestionFromAssertion(t *testing.T) {
	service := createTestService()

	tests := []struct {
		assertionType string
		expected      string
	}{
		{"data_match", "Check API response format and UI data binding"},
		{"status_match", "Verify API endpoint status codes"},
		{"timing_match", "Optimize API response time or adjust timeout values"},
		{"ui_state", "Check UI component state management"},
		{"unknown", "Review assertion configuration and expected values"},
	}

	for _, tt := range tests {
		t.Run(tt.assertionType, func(t *testing.T) {
			assertion := models.SyncAssertion{Type: tt.assertionType}
			result := &models.SyncAssertionResult{}
			suggestion := service.getSuggestionFromAssertion(assertion, result)
			assert.Equal(t, tt.expected, suggestion)
		})
	}
}

func TestTestService_AnalyzeSyncIssues(t *testing.T) {
	service := createTestService()

	run := &TestRun{
		ID: "test-run",
		Results: &models.TestResults{
			RunID:      "test-run",
			SyncIssues: make([]models.SyncIssue, 0),
			Results: []models.TestCase{
				{
					Name:     "Test 1",
					Status:   "passed",
					ErrorMsg: "",
				},
				{
					Name:     "Test 2",
					Status:   "failed",
					ErrorMsg: "Timeout waiting for response",
				},
				{
					Name:     "Test 3",
					Status:   "failed",
					ErrorMsg: "Data mismatch in response",
				},
			},
		},
	}

	service.analyzeSyncIssues(run)

	// Should find at least 2 issues (timeout and data sync)
	assert.GreaterOrEqual(t, len(run.Results.SyncIssues), 2)

	// Check that we have timeout and data sync issues
	var hasTimeoutIssue, hasDataSyncIssue bool
	for _, issue := range run.Results.SyncIssues {
		if issue.Type == "timeout_mismatch" {
			hasTimeoutIssue = true
			assert.Equal(t, "warning", issue.Severity)
			assert.Contains(t, issue.Description, "Test 2")
		}
		if issue.Type == "data_sync_error" {
			hasDataSyncIssue = true
			assert.Equal(t, "critical", issue.Severity)
		}
	}

	assert.True(t, hasTimeoutIssue, "Should have timeout issue")
	assert.True(t, hasDataSyncIssue, "Should have data sync issue")
}

func TestTestService_GetStatus(t *testing.T) {
	service := createTestService()

	// Add some test data
	service.mu.Lock()
	service.activeRuns["run-1"] = &TestRun{ID: "run-1"}
	service.activeRuns["run-2"] = &TestRun{ID: "run-2"}
	service.runHistory = []models.TestResults{
		{RunID: "completed-1"},
		{RunID: "completed-2"},
		{RunID: "completed-3"},
	}
	service.mu.Unlock()

	status := service.GetStatus()

	assert.Equal(t, 2, status["active_runs"])
	assert.Equal(t, 3, status["history_count"])

	frameworks, ok := status["supported_frameworks"].([]string)
	assert.True(t, ok)
	assert.Contains(t, frameworks, "cypress")
	assert.Contains(t, frameworks, "playwright")
	assert.Contains(t, frameworks, "jest")
	assert.Contains(t, frameworks, "vitest")
}

func TestTestService_ParseSimpleTestOutput(t *testing.T) {
	service := createTestService()

	run := &TestRun{
		Results: &models.TestResults{
			Results: make([]models.TestCase, 0),
		},
	}

	// Test output with passes and failures
	output := `
	✓ Test 1 passed
	✓ Test 2 passed
	✗ Test 3 failed
	✓ Test 4 passed
	`

	err := service.parseSimpleTestOutput(run, output)
	require.NoError(t, err)

	assert.Equal(t, 4, run.Results.TotalTests)
	assert.Equal(t, 3, run.Results.PassedTests)
	assert.Equal(t, 1, run.Results.FailedTests)
}

func TestTestService_MoveToHistory(t *testing.T) {
	service := createTestService()

	// Create a test run
	run := &TestRun{
		ID: "test-run",
		Results: &models.TestResults{
			RunID:  "test-run",
			Status: "completed",
		},
	}

	// Add to active runs
	service.mu.Lock()
	service.activeRuns["test-run"] = run
	service.mu.Unlock()

	// Move to history
	service.moveToHistory(run)

	// Verify it's removed from active runs
	service.mu.RLock()
	_, exists := service.activeRuns["test-run"]
	historyCount := len(service.runHistory)
	service.mu.RUnlock()

	assert.False(t, exists)
	assert.Equal(t, 1, historyCount)
	assert.Equal(t, "test-run", service.runHistory[0].RunID)
}

func TestTestService_MoveToHistory_MaxHistoryLimit(t *testing.T) {
	service := createTestService()
	service.maxHistory = 2 // Set small limit for testing

	// Fill history to max
	service.mu.Lock()
	service.runHistory = []models.TestResults{
		{RunID: "old-run-1"},
		{RunID: "old-run-2"},
	}
	service.mu.Unlock()

	// Create new run
	run := &TestRun{
		ID: "new-run",
		Results: &models.TestResults{
			RunID:  "new-run",
			Status: "completed",
		},
	}

	service.mu.Lock()
	service.activeRuns["new-run"] = run
	service.mu.Unlock()

	// Move to history
	service.moveToHistory(run)

	// Verify history is trimmed
	service.mu.RLock()
	historyCount := len(service.runHistory)
	service.mu.RUnlock()

	assert.Equal(t, 2, historyCount)
	assert.Equal(t, "old-run-2", service.runHistory[0].RunID)
	assert.Equal(t, "new-run", service.runHistory[1].RunID)
}

// Helper function to create a test service
func createTestService() *TestService {
	cfg := &config.Config{
		CypressBaseURL:    "http://localhost:3000",
		PlaywrightBaseURL: "http://localhost:3000",
	}

	wsHub := &models.WSHub{
		Clients:    make(map[*models.WSClient]bool),
		Broadcast:  make(chan models.WSMessage, 256),
		Register:   make(chan *models.WSClient),
		Unregister: make(chan *models.WSClient),
	}

	return NewTestService(cfg, wsHub)
}
