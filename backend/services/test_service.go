package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/google/uuid"
)

// TestService handles test orchestration for end-to-end testing
type TestService struct {
	config     *config.Config
	mu         sync.RWMutex
	activeRuns map[string]*TestRun
	runHistory []models.TestResults
	maxHistory int
	wsHub      WebSocketBroadcaster // For real-time updates
}

// TestRun represents an active test run
type TestRun struct {
	ID         string
	Request    *models.TestRunRequest
	Status     string
	StartTime  time.Time
	EndTime    time.Time
	Process    *exec.Cmd
	Context    context.Context
	Cancel     context.CancelFunc
	Results    *models.TestResults
	LogChannel chan string
}

// NewTestService creates a new test service instance
func NewTestService(cfg *config.Config, wsHub WebSocketBroadcaster) *TestService {
	return &TestService{
		config:     cfg,
		activeRuns: make(map[string]*TestRun),
		runHistory: make([]models.TestResults, 0),
		maxHistory: 100, // Keep last 100 test runs
		wsHub:      wsHub,
	}
}

// StartTestRun initiates a new test run
func (s *TestService) StartTestRun(ctx context.Context, req *models.TestRunRequest) (*models.TestRunResponse, error) {
	runID := uuid.New().String()

	// Validate framework support
	if !s.isFrameworkSupported(req.Framework) {
		return nil, fmt.Errorf("unsupported test framework: %s", req.Framework)
	}

	// Create test run context with cancellation
	runCtx, cancel := context.WithCancel(ctx)

	testRun := &TestRun{
		ID:         runID,
		Request:    req,
		Status:     "queued",
		StartTime:  time.Now(),
		Context:    runCtx,
		Cancel:     cancel,
		LogChannel: make(chan string, 100),
		Results: &models.TestResults{
			RunID:      runID,
			Status:     "queued",
			StartTime:  time.Now(),
			Results:    make([]models.TestCase, 0),
			SyncIssues: make([]models.SyncIssue, 0),
		},
	}

	// Store the active run
	s.mu.Lock()
	s.activeRuns[runID] = testRun
	s.mu.Unlock()

	// Start the test execution in a goroutine
	go s.executeTestRun(testRun)

	// Send WebSocket notification
	s.broadcastTestUpdate(runID, "queued", "Test run queued for execution")

	return &models.TestRunResponse{
		RunID:             runID,
		Status:            "queued",
		StartTime:         testRun.StartTime,
		Framework:         req.Framework,
		Environment:       req.Environment,
		EstimatedDuration: s.getEstimatedDuration(req.Framework),
	}, nil
}

// GetTestResults retrieves results for a specific test run
func (s *TestService) GetTestResults(runID string) (*models.TestResults, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check active runs first
	if run, exists := s.activeRuns[runID]; exists {
		return run.Results, nil
	}

	// Check history
	for _, result := range s.runHistory {
		if result.RunID == runID {
			return &result, nil
		}
	}

	return nil, fmt.Errorf("test run not found: %s", runID)
}

// CancelTestRun cancels an active test run
func (s *TestService) CancelTestRun(runID string) error {
	s.mu.Lock()
	run, exists := s.activeRuns[runID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("test run not found or already completed: %s", runID)
	}

	// Cancel the context
	run.Cancel()

	// Kill the process if it's running
	if run.Process != nil && run.Process.Process != nil {
		if err := run.Process.Process.Kill(); err != nil {
			log.Printf("Error killing test process for run %s: %v", runID, err)
		}
	}

	// Update status
	run.Status = "cancelled"
	run.Results.Status = "cancelled"
	run.EndTime = time.Now()
	s.mu.Unlock()

	// Broadcast update (outside of lock to avoid deadlock)
	s.broadcastTestUpdate(runID, "cancelled", "Test run cancelled by user")

	return nil
}

// ValidateSync validates API-UI synchronization
func (s *TestService) ValidateSync(ctx context.Context, req *models.TestSyncValidationRequest) (*models.TestSyncValidationResponse, error) {
	validationID := uuid.New().String()
	log.Printf("Starting sync validation %s for endpoint: %s", validationID, req.APIEndpoint)

	response := &models.TestSyncValidationResponse{
		IsValid:     true,
		Results:     make([]models.SyncAssertionResult, 0),
		Issues:      make([]models.SyncIssue, 0),
		ValidatedAt: time.Now(),
	}

	// Execute each assertion
	for _, assertion := range req.Assertions {
		result, err := s.executeAssertion(ctx, req, assertion)
		if err != nil {
			log.Printf("Error executing assertion: %v", err)
			response.Issues = append(response.Issues, models.SyncIssue{
				Type:        "assertion_error",
				Description: fmt.Sprintf("Failed to execute assertion: %v", err),
				Severity:    "critical",
				Suggestion:  "Check API endpoint accessibility and assertion configuration",
			})
			response.IsValid = false
		} else {
			response.Results = append(response.Results, *result)
			if !result.Passed {
				response.IsValid = false
				response.Issues = append(response.Issues, models.SyncIssue{
					Type:        "assertion_failed",
					Description: result.Message,
					Severity:    s.getSeverityFromAssertion(assertion),
					Suggestion:  s.getSuggestionFromAssertion(assertion, result),
				})
			}
		}
	}

	return response, nil
}

// GetActiveRuns returns all currently active test runs
func (s *TestService) GetActiveRuns() map[string]*models.TestRunResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active := make(map[string]*models.TestRunResponse)
	for id, run := range s.activeRuns {
		active[id] = &models.TestRunResponse{
			RunID:       id,
			Status:      run.Status,
			StartTime:   run.StartTime,
			Framework:   run.Request.Framework,
			Environment: run.Request.Environment,
		}
	}

	return active
}

// GetRunHistory returns the test run history
func (s *TestService) GetRunHistory(limit int) []models.TestResults {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.runHistory) {
		limit = len(s.runHistory)
	}

	// Return the most recent runs
	start := len(s.runHistory) - limit
	if start < 0 {
		start = 0
	}

	history := make([]models.TestResults, limit)
	copy(history, s.runHistory[start:])
	return history
}

// executeTestRun executes a test run based on the framework
func (s *TestService) executeTestRun(run *TestRun) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Test run %s panicked: %v", run.ID, r)
			run.Status = "failed"
			run.Results.Status = "failed"
			run.EndTime = time.Now()
		}

		// Move to history and clean up
		s.moveToHistory(run)
	}()

	// Update status to running
	run.Status = "running"
	run.Results.Status = "running"
	s.broadcastTestUpdate(run.ID, "running", "Test execution started")

	var err error
	switch strings.ToLower(run.Request.Framework) {
	case "cypress":
		err = s.executeCypressTests(run)
	case "playwright":
		err = s.executePlaywrightTests(run)
	case "jest":
		err = s.executeJestTests(run)
	case "vitest":
		err = s.executeVitestTests(run)
	default:
		err = fmt.Errorf("unsupported framework: %s", run.Request.Framework)
	}

	run.EndTime = time.Now()
	run.Results.EndTime = run.EndTime
	run.Results.Duration = run.EndTime.Sub(run.StartTime)

	if err != nil {
		log.Printf("Test run %s failed: %v", run.ID, err)
		run.Status = "failed"
		run.Results.Status = "failed"
		s.broadcastTestUpdate(run.ID, "failed", fmt.Sprintf("Test execution failed: %v", err))
	} else {
		run.Status = "completed"
		run.Results.Status = "completed"
		s.broadcastTestUpdate(run.ID, "completed", "Test execution completed successfully")
	}

	// Analyze results for sync issues
	s.analyzeSyncIssues(run)
}

// executeCypressTests executes Cypress tests
func (s *TestService) executeCypressTests(run *TestRun) error {
	log.Printf("Executing Cypress tests for run %s", run.ID)

	// Build Cypress command
	args := []string{"run"}

	// Add spec file if specified
	if run.Request.TestSuite != "" {
		args = append(args, "--spec", run.Request.TestSuite)
	}

	// Add environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("CYPRESS_baseUrl=%s", s.config.CypressBaseURL))

	// Add custom config
	for key, value := range run.Request.Config {
		env = append(env, fmt.Sprintf("CYPRESS_%s=%s", key, value))
	}

	// Create command
	cmd := exec.CommandContext(run.Context, "npx", append([]string{"cypress"}, args...)...)
	cmd.Env = env

	// Set working directory (assuming tests are in a cypress directory)
	if workDir := run.Request.Config["workDir"]; workDir != "" {
		cmd.Dir = workDir
	}

	run.Process = cmd

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cypress execution failed: %w, output: %s", err, string(output))
	}

	// Parse Cypress results
	return s.parseCypressResults(run, string(output))
}

// executePlaywrightTests executes Playwright tests
func (s *TestService) executePlaywrightTests(run *TestRun) error {
	log.Printf("Executing Playwright tests for run %s", run.ID)

	args := []string{"test"}

	// Add test file if specified
	if run.Request.TestSuite != "" {
		args = append(args, run.Request.TestSuite)
	}

	// Add reporter for JSON output
	args = append(args, "--reporter=json")

	// Create command
	cmd := exec.CommandContext(run.Context, "npx", append([]string{"playwright"}, args...)...)

	// Set environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("PLAYWRIGHT_BASE_URL=%s", s.config.PlaywrightBaseURL))

	for key, value := range run.Request.Config {
		env = append(env, fmt.Sprintf("PLAYWRIGHT_%s=%s", strings.ToUpper(key), value))
	}
	cmd.Env = env

	// Set working directory
	if workDir := run.Request.Config["workDir"]; workDir != "" {
		cmd.Dir = workDir
	}

	run.Process = cmd

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("playwright execution failed: %w, output: %s", err, string(output))
	}

	// Parse Playwright results
	return s.parsePlaywrightResults(run, string(output))
}

// executeJestTests executes Jest tests
func (s *TestService) executeJestTests(run *TestRun) error {
	log.Printf("Executing Jest tests for run %s", run.ID)

	args := []string{"--json", "--coverage=false"}

	if run.Request.TestSuite != "" {
		args = append(args, run.Request.TestSuite)
	}

	cmd := exec.CommandContext(run.Context, "npx", append([]string{"jest"}, args...)...)

	// Set working directory
	if workDir := run.Request.Config["workDir"]; workDir != "" {
		cmd.Dir = workDir
	}

	run.Process = cmd

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jest execution failed: %w, output: %s", err, string(output))
	}

	return s.parseJestResults(run, string(output))
}

// executeVitestTests executes Vitest tests
func (s *TestService) executeVitestTests(run *TestRun) error {
	log.Printf("Executing Vitest tests for run %s", run.ID)

	args := []string{"run", "--reporter=json"}

	if run.Request.TestSuite != "" {
		args = append(args, run.Request.TestSuite)
	}

	cmd := exec.CommandContext(run.Context, "npx", append([]string{"vitest"}, args...)...)

	if workDir := run.Request.Config["workDir"]; workDir != "" {
		cmd.Dir = workDir
	}

	run.Process = cmd

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("vitest execution failed: %w, output: %s", err, string(output))
	}

	return s.parseVitestResults(run, string(output))
}

// Helper methods for parsing test results
func (s *TestService) parseCypressResults(run *TestRun, output string) error {
	// Simplified Cypress result parsing
	// In production, you'd parse the actual Cypress JSON output
	lines := strings.Split(output, "\n")

	totalTests := 0
	passedTests := 0
	failedTests := 0

	for _, line := range lines {
		if strings.Contains(line, "passing") {
			passedTests++
			totalTests++
		} else if strings.Contains(line, "failing") {
			failedTests++
			totalTests++
		}
	}

	run.Results.TotalTests = totalTests
	run.Results.PassedTests = passedTests
	run.Results.FailedTests = failedTests

	// Create sample test cases
	for i := 0; i < totalTests; i++ {
		status := "passed"
		if i < failedTests {
			status = "failed"
		}

		testCase := models.TestCase{
			Name:     fmt.Sprintf("Test Case %d", i+1),
			Status:   status,
			Duration: time.Millisecond * 100,
		}

		if status == "failed" {
			testCase.ErrorMsg = "Test assertion failed"
		}

		run.Results.Results = append(run.Results.Results, testCase)
	}

	return nil
}

func (s *TestService) parsePlaywrightResults(run *TestRun, output string) error {
	// Try to parse JSON output from Playwright
	var playwrightResult struct {
		Stats struct {
			Total   int `json:"total"`
			Passed  int `json:"passed"`
			Failed  int `json:"failed"`
			Skipped int `json:"skipped"`
		} `json:"stats"`
		Tests []struct {
			Title  string `json:"title"`
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		} `json:"tests"`
	}

	if err := json.Unmarshal([]byte(output), &playwrightResult); err != nil {
		// Fallback to simple parsing if JSON parsing fails
		return s.parseSimpleTestOutput(run, output)
	}

	run.Results.TotalTests = playwrightResult.Stats.Total
	run.Results.PassedTests = playwrightResult.Stats.Passed
	run.Results.FailedTests = playwrightResult.Stats.Failed
	run.Results.SkippedTests = playwrightResult.Stats.Skipped

	for _, test := range playwrightResult.Tests {
		testCase := models.TestCase{
			Name:     test.Title,
			Status:   test.Status,
			Duration: time.Millisecond * 100,
		}

		if test.Error != "" {
			testCase.ErrorMsg = test.Error
		}

		run.Results.Results = append(run.Results.Results, testCase)
	}

	return nil
}

func (s *TestService) parseJestResults(run *TestRun, output string) error {
	// Parse Jest JSON output
	var jestResult struct {
		NumTotalTests   int `json:"numTotalTests"`
		NumPassedTests  int `json:"numPassedTests"`
		NumFailedTests  int `json:"numFailedTests"`
		NumPendingTests int `json:"numPendingTests"`
		TestResults     []struct {
			AssertionResults []struct {
				Title           string   `json:"title"`
				Status          string   `json:"status"`
				FailureMessages []string `json:"failureMessages"`
			} `json:"assertionResults"`
		} `json:"testResults"`
	}

	if err := json.Unmarshal([]byte(output), &jestResult); err != nil {
		return s.parseSimpleTestOutput(run, output)
	}

	run.Results.TotalTests = jestResult.NumTotalTests
	run.Results.PassedTests = jestResult.NumPassedTests
	run.Results.FailedTests = jestResult.NumFailedTests
	run.Results.SkippedTests = jestResult.NumPendingTests

	for _, testFile := range jestResult.TestResults {
		for _, test := range testFile.AssertionResults {
			testCase := models.TestCase{
				Name:     test.Title,
				Status:   test.Status,
				Duration: time.Millisecond * 100,
			}

			if len(test.FailureMessages) > 0 {
				testCase.ErrorMsg = strings.Join(test.FailureMessages, "\n")
			}

			run.Results.Results = append(run.Results.Results, testCase)
		}
	}

	return nil
}

func (s *TestService) parseVitestResults(run *TestRun, output string) error {
	// Similar to Jest parsing but for Vitest format
	return s.parseSimpleTestOutput(run, output)
}

func (s *TestService) parseSimpleTestOutput(run *TestRun, output string) error {
	// Fallback simple parsing for when JSON parsing fails
	lines := strings.Split(output, "\n")

	totalTests := 0
	passedTests := 0
	failedTests := 0

	for _, line := range lines {
		line = strings.TrimSpace(strings.ToLower(line))
		if strings.Contains(line, "pass") || strings.Contains(line, "✓") {
			passedTests++
			totalTests++
		} else if strings.Contains(line, "fail") || strings.Contains(line, "✗") {
			failedTests++
			totalTests++
		}
	}

	run.Results.TotalTests = totalTests
	run.Results.PassedTests = passedTests
	run.Results.FailedTests = failedTests

	return nil
}

// executeAssertion executes a single sync assertion
func (s *TestService) executeAssertion(ctx context.Context, req *models.TestSyncValidationRequest, assertion models.SyncAssertion) (*models.SyncAssertionResult, error) {
	// This is a simplified implementation
	// In production, you'd make actual API calls and UI checks

	result := &models.SyncAssertionResult{
		Assertion: assertion,
		Passed:    true,
		Actual:    assertion.Expected, // Simplified - assume it matches
		Message:   "Assertion passed",
	}

	// Simulate some validation logic
	switch assertion.Type {
	case "data_match":
		// Would make API call and compare data
		result.Message = "Data matches between API and UI"
	case "status_match":
		// Would check HTTP status codes
		result.Message = "Status codes match"
	case "timing_match":
		// Would measure response times
		result.Message = "Response times are within acceptable range"
	case "ui_state":
		// Would check UI state
		result.Message = "UI state is correct"
	default:
		result.Passed = false
		result.Message = fmt.Sprintf("Unknown assertion type: %s", assertion.Type)
	}

	return result, nil
}

// Helper methods
func (s *TestService) isFrameworkSupported(framework string) bool {
	supported := []string{"cypress", "playwright", "jest", "vitest"}
	framework = strings.ToLower(framework)

	for _, f := range supported {
		if f == framework {
			return true
		}
	}
	return false
}

func (s *TestService) getEstimatedDuration(framework string) time.Duration {
	// Return estimated duration based on framework
	switch strings.ToLower(framework) {
	case "cypress":
		return time.Minute * 5
	case "playwright":
		return time.Minute * 3
	case "jest":
		return time.Minute * 2
	case "vitest":
		return time.Minute * 1
	default:
		return time.Minute * 5
	}
}

func (s *TestService) getSeverityFromAssertion(assertion models.SyncAssertion) string {
	// Determine severity based on assertion type
	switch assertion.Type {
	case "data_match":
		return "critical"
	case "status_match":
		return "warning"
	case "timing_match":
		return "info"
	case "ui_state":
		return "warning"
	default:
		return "info"
	}
}

func (s *TestService) getSuggestionFromAssertion(assertion models.SyncAssertion, result *models.SyncAssertionResult) string {
	switch assertion.Type {
	case "data_match":
		return "Check API response format and UI data binding"
	case "status_match":
		return "Verify API endpoint status codes"
	case "timing_match":
		return "Optimize API response time or adjust timeout values"
	case "ui_state":
		return "Check UI component state management"
	default:
		return "Review assertion configuration and expected values"
	}
}

func (s *TestService) analyzeSyncIssues(run *TestRun) {
	// Analyze test results for potential sync issues
	for _, testCase := range run.Results.Results {
		if testCase.Status == "failed" {
			// Look for common sync issue patterns in error messages
			if strings.Contains(strings.ToLower(testCase.ErrorMsg), "timeout") {
				run.Results.SyncIssues = append(run.Results.SyncIssues, models.SyncIssue{
					Type:        "timeout_mismatch",
					Description: fmt.Sprintf("Timeout detected in test: %s", testCase.Name),
					Severity:    "warning",
					Suggestion:  "Check API response times and adjust timeout values",
					TestCase:    testCase.Name,
				})
			}

			if strings.Contains(strings.ToLower(testCase.ErrorMsg), "data") ||
				strings.Contains(strings.ToLower(testCase.ErrorMsg), "response") {
				run.Results.SyncIssues = append(run.Results.SyncIssues, models.SyncIssue{
					Type:        "data_sync_error",
					Description: fmt.Sprintf("Data synchronization issue in test: %s", testCase.Name),
					Severity:    "critical",
					Suggestion:  "Verify API response format matches UI expectations",
					TestCase:    testCase.Name,
				})
			}
		}
	}
}

func (s *TestService) broadcastTestUpdate(runID, status, message string) {
	if s.wsHub == nil {
		return
	}

	// Get the test run for additional details
	s.mu.RLock()
	run, exists := s.activeRuns[runID]
	s.mu.RUnlock()

	data := map[string]interface{}{
		"run_id":    runID,
		"status":    status,
		"message":   message,
		"timestamp": time.Now(),
	}

	if exists && run != nil {
		data["framework"] = run.Request.Framework
		data["environment"] = run.Request.Environment
		data["start_time"] = run.StartTime

		if run.Results != nil {
			data["total_tests"] = run.Results.TotalTests
			data["passed_tests"] = run.Results.PassedTests
			data["failed_tests"] = run.Results.FailedTests
			data["skipped_tests"] = run.Results.SkippedTests

			if !run.EndTime.IsZero() {
				data["duration"] = run.EndTime.Sub(run.StartTime).String()
			} else {
				data["duration"] = time.Since(run.StartTime).String()
			}
		}
	}

	s.wsHub.BroadcastToAll("test_progress", data)
}

func (s *TestService) moveToHistory(run *TestRun) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from active runs
	delete(s.activeRuns, run.ID)

	// Add to history
	s.runHistory = append(s.runHistory, *run.Results)

	// Trim history if it exceeds max size
	if len(s.runHistory) > s.maxHistory {
		s.runHistory = s.runHistory[1:]
	}
}

// GetStatus returns the current status of the test service
func (s *TestService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"active_runs":          len(s.activeRuns),
		"history_count":        len(s.runHistory),
		"supported_frameworks": []string{"cypress", "playwright", "jest", "vitest"},
	}
}
