package models

import "time"

// TestRunRequest represents a request to run tests
type TestRunRequest struct {
	Framework   string            `json:"framework" validate:"required,oneof=cypress playwright jest vitest"`
	TestSuite   string            `json:"test_suite" validate:"required,min=1"`
	Environment string            `json:"environment" validate:"required,min=1"`
	Config      map[string]string `json:"config"`
	Tags        []string          `json:"tags"`
}

// TestRunResponse represents the response when starting a test run
type TestRunResponse struct {
	RunID             string        `json:"run_id" validate:"required"`
	Status            string        `json:"status" validate:"required,oneof=queued running completed failed cancelled"`
	StartTime         time.Time     `json:"start_time"`
	Framework         string        `json:"framework"`
	Environment       string        `json:"environment"`
	EstimatedDuration time.Duration `json:"estimated_duration"`
}

// TestResults represents the complete results of a test run
type TestResults struct {
	RunID        string        `json:"run_id" validate:"required"`
	Status       string        `json:"status" validate:"required,oneof=running completed failed cancelled"`
	TotalTests   int           `json:"total_tests" validate:"min=0"`
	PassedTests  int           `json:"passed_tests" validate:"min=0"`
	FailedTests  int           `json:"failed_tests" validate:"min=0"`
	SkippedTests int           `json:"skipped_tests" validate:"min=0"`
	Duration     time.Duration `json:"duration"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Results      []TestCase    `json:"results"`
	SyncIssues   []SyncIssue   `json:"sync_issues"`
	Coverage     *TestCoverage `json:"coverage,omitempty"`
}

// TestCase represents an individual test case result
type TestCase struct {
	Name        string        `json:"name" validate:"required,min=1"`
	Status      string        `json:"status" validate:"required,oneof=passed failed skipped"`
	Duration    time.Duration `json:"duration"`
	ErrorMsg    string        `json:"error_msg,omitempty"`
	StackTrace  string        `json:"stack_trace,omitempty"`
	Screenshots []string      `json:"screenshots,omitempty"`
	Steps       []TestStep    `json:"steps,omitempty"`
	Tags        []string      `json:"tags"`
}

// TestStep represents a step within a test case
type TestStep struct {
	Name        string        `json:"name" validate:"required"`
	Status      string        `json:"status" validate:"required,oneof=passed failed skipped"`
	Duration    time.Duration `json:"duration"`
	Description string        `json:"description"`
	ErrorMsg    string        `json:"error_msg,omitempty"`
}

// SyncIssue represents a synchronization issue found during testing
type SyncIssue struct {
	Type        string `json:"type" validate:"required,oneof=api_mismatch ui_inconsistency data_sync_error timeout_mismatch"`
	Description string `json:"description" validate:"required"`
	Severity    string `json:"severity" validate:"required,oneof=critical warning info"`
	Suggestion  string `json:"suggestion"`
	TestCase    string `json:"test_case"`
	Location    string `json:"location"`
}

// TestCoverage represents test coverage information
type TestCoverage struct {
	Lines      CoverageMetric `json:"lines"`
	Functions  CoverageMetric `json:"functions"`
	Branches   CoverageMetric `json:"branches"`
	Statements CoverageMetric `json:"statements"`
}

// CoverageMetric represents a coverage metric
type CoverageMetric struct {
	Total   int     `json:"total" validate:"min=0"`
	Covered int     `json:"covered" validate:"min=0"`
	Percent float64 `json:"percent" validate:"min=0,max=100"`
}

// TestSyncValidationRequest represents a request to validate API-UI synchronization
type TestSyncValidationRequest struct {
	APIEndpoint string            `json:"api_endpoint" validate:"required,url"`
	UIComponent string            `json:"ui_component" validate:"required,min=1"`
	TestData    interface{}       `json:"test_data"`
	Assertions  []SyncAssertion   `json:"assertions" validate:"required,min=1"`
	Config      map[string]string `json:"config"`
}

// SyncAssertion represents an assertion for sync validation
type SyncAssertion struct {
	Type        string      `json:"type" validate:"required,oneof=data_match status_match timing_match ui_state"`
	Field       string      `json:"field"`
	Expected    interface{} `json:"expected"`
	Operator    string      `json:"operator" validate:"required,oneof=equals not_equals contains greater_than less_than exists"`
	Description string      `json:"description"`
}

// TestSyncValidationResponse represents the response from sync validation
type TestSyncValidationResponse struct {
	IsValid     bool                  `json:"is_valid"`
	Results     []SyncAssertionResult `json:"results"`
	Issues      []SyncIssue           `json:"issues"`
	Performance *PerformanceMetrics   `json:"performance,omitempty"`
	ValidatedAt time.Time             `json:"validated_at"`
}

// SyncAssertionResult represents the result of a sync assertion
type SyncAssertionResult struct {
	Assertion SyncAssertion `json:"assertion"`
	Passed    bool          `json:"passed"`
	Actual    interface{}   `json:"actual"`
	Message   string        `json:"message"`
}

// PerformanceMetrics represents performance metrics for sync validation
type PerformanceMetrics struct {
	APIResponseTime  time.Duration `json:"api_response_time"`
	UIRenderTime     time.Duration `json:"ui_render_time"`
	TotalSyncTime    time.Duration `json:"total_sync_time"`
	DataTransferSize int64         `json:"data_transfer_size"`
}
