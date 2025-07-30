package models

import (
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
)

func TestTestRunRequestValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		request   TestRunRequest
		wantValid bool
		wantError string
	}{
		{
			name: "valid Cypress test request",
			request: TestRunRequest{
				Framework:   "cypress",
				TestSuite:   "e2e/user-auth.spec.js",
				Environment: "staging",
				Config: map[string]string{
					"baseUrl": "http://localhost:3000",
					"timeout": "10000",
				},
				Tags: []string{"auth", "critical"},
			},
			wantValid: true,
		},
		{
			name: "valid Playwright test request",
			request: TestRunRequest{
				Framework:   "playwright",
				TestSuite:   "tests/integration/api.spec.ts",
				Environment: "development",
				Config: map[string]string{
					"headless": "true",
				},
				Tags: []string{"api", "integration"},
			},
			wantValid: true,
		},
		{
			name: "invalid framework",
			request: TestRunRequest{
				Framework:   "invalid_framework",
				TestSuite:   "test.spec.js",
				Environment: "development",
				Config:      map[string]string{},
				Tags:        []string{},
			},
			wantValid: false,
			wantError: "framework",
		},
		{
			name: "missing test suite",
			request: TestRunRequest{
				Framework:   "cypress",
				TestSuite:   "",
				Environment: "development",
				Config:      map[string]string{},
				Tags:        []string{},
			},
			wantValid: false,
			wantError: "test_suite",
		},
		{
			name: "missing environment",
			request: TestRunRequest{
				Framework:   "jest",
				TestSuite:   "unit/utils.test.js",
				Environment: "",
				Config:      map[string]string{},
				Tags:        []string{},
			},
			wantValid: false,
			wantError: "environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.request)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid request, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid request, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestTestRunResponseValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		response  TestRunResponse
		wantValid bool
		wantError string
	}{
		{
			name: "valid queued test response",
			response: TestRunResponse{
				RunID:             "run-123",
				Status:            "queued",
				StartTime:         time.Now(),
				Framework:         "cypress",
				Environment:       "staging",
				EstimatedDuration: 5 * time.Minute,
			},
			wantValid: true,
		},
		{
			name: "valid running test response",
			response: TestRunResponse{
				RunID:             "run-456",
				Status:            "running",
				StartTime:         time.Now(),
				Framework:         "playwright",
				Environment:       "production",
				EstimatedDuration: 10 * time.Minute,
			},
			wantValid: true,
		},
		{
			name: "missing run ID",
			response: TestRunResponse{
				RunID:             "",
				Status:            "queued",
				StartTime:         time.Now(),
				Framework:         "cypress",
				Environment:       "staging",
				EstimatedDuration: 5 * time.Minute,
			},
			wantValid: false,
			wantError: "run_id",
		},
		{
			name: "invalid status",
			response: TestRunResponse{
				RunID:             "run-789",
				Status:            "invalid_status",
				StartTime:         time.Now(),
				Framework:         "cypress",
				Environment:       "staging",
				EstimatedDuration: 5 * time.Minute,
			},
			wantValid: false,
			wantError: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.response)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid response, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid response, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestTestResultsValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		results   TestResults
		wantValid bool
		wantError string
	}{
		{
			name: "valid completed test results",
			results: TestResults{
				RunID:        "run-123",
				Status:       "completed",
				TotalTests:   10,
				PassedTests:  8,
				FailedTests:  2,
				SkippedTests: 0,
				Duration:     5 * time.Minute,
				StartTime:    time.Now().Add(-5 * time.Minute),
				EndTime:      time.Now(),
				Results: []TestCase{
					{
						Name:     "should login successfully",
						Status:   "passed",
						Duration: 30 * time.Second,
						Tags:     []string{"auth"},
					},
				},
				SyncIssues: []SyncIssue{},
			},
			wantValid: true,
		},
		{
			name: "valid failed test results",
			results: TestResults{
				RunID:        "run-456",
				Status:       "failed",
				TotalTests:   5,
				PassedTests:  3,
				FailedTests:  2,
				SkippedTests: 0,
				Duration:     3 * time.Minute,
				StartTime:    time.Now().Add(-3 * time.Minute),
				EndTime:      time.Now(),
				Results: []TestCase{
					{
						Name:       "should handle API error",
						Status:     "failed",
						Duration:   45 * time.Second,
						ErrorMsg:   "Expected 200 but got 500",
						StackTrace: "Error at line 15...",
						Tags:       []string{"api", "error"},
					},
				},
				SyncIssues: []SyncIssue{
					{
						Type:        "api_mismatch",
						Description: "API response format changed",
						Severity:    "critical",
						Suggestion:  "Update API contract",
						TestCase:    "should handle API error",
						Location:    "api.spec.js:15",
					},
				},
			},
			wantValid: true,
		},
		{
			name: "missing run ID",
			results: TestResults{
				RunID:        "",
				Status:       "completed",
				TotalTests:   1,
				PassedTests:  1,
				FailedTests:  0,
				SkippedTests: 0,
				Duration:     1 * time.Minute,
				StartTime:    time.Now().Add(-1 * time.Minute),
				EndTime:      time.Now(),
				Results:      []TestCase{},
				SyncIssues:   []SyncIssue{},
			},
			wantValid: false,
			wantError: "run_id",
		},
		{
			name: "invalid status",
			results: TestResults{
				RunID:        "run-789",
				Status:       "invalid_status",
				TotalTests:   1,
				PassedTests:  1,
				FailedTests:  0,
				SkippedTests: 0,
				Duration:     1 * time.Minute,
				StartTime:    time.Now().Add(-1 * time.Minute),
				EndTime:      time.Now(),
				Results:      []TestCase{},
				SyncIssues:   []SyncIssue{},
			},
			wantValid: false,
			wantError: "status",
		},
		{
			name: "negative total tests",
			results: TestResults{
				RunID:        "run-999",
				Status:       "completed",
				TotalTests:   -1,
				PassedTests:  0,
				FailedTests:  0,
				SkippedTests: 0,
				Duration:     1 * time.Minute,
				StartTime:    time.Now().Add(-1 * time.Minute),
				EndTime:      time.Now(),
				Results:      []TestCase{},
				SyncIssues:   []SyncIssue{},
			},
			wantValid: false,
			wantError: "total_tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.results)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid results, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid results, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestTestCaseValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		testCase  TestCase
		wantValid bool
		wantError string
	}{
		{
			name: "valid passed test case",
			testCase: TestCase{
				Name:     "should render homepage correctly",
				Status:   "passed",
				Duration: 2 * time.Second,
				Tags:     []string{"ui", "smoke"},
			},
			wantValid: true,
		},
		{
			name: "valid failed test case with error",
			testCase: TestCase{
				Name:       "should handle form submission",
				Status:     "failed",
				Duration:   5 * time.Second,
				ErrorMsg:   "Element not found: #submit-button",
				StackTrace: "Error at line 25 in form.spec.js",
				Screenshots: []string{
					"screenshot-1.png",
					"screenshot-2.png",
				},
				Tags: []string{"form", "ui"},
			},
			wantValid: true,
		},
		{
			name: "missing test name",
			testCase: TestCase{
				Name:     "",
				Status:   "passed",
				Duration: 1 * time.Second,
				Tags:     []string{},
			},
			wantValid: false,
			wantError: "name",
		},
		{
			name: "invalid status",
			testCase: TestCase{
				Name:     "test case",
				Status:   "invalid_status",
				Duration: 1 * time.Second,
				Tags:     []string{},
			},
			wantValid: false,
			wantError: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.testCase)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid test case, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid test case, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestSyncIssueValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		issue     SyncIssue
		wantValid bool
		wantError string
	}{
		{
			name: "valid API mismatch issue",
			issue: SyncIssue{
				Type:        "api_mismatch",
				Description: "Response schema changed unexpectedly",
				Severity:    "critical",
				Suggestion:  "Update API contract and tests",
				TestCase:    "should fetch user data",
				Location:    "user.spec.js:42",
			},
			wantValid: true,
		},
		{
			name: "valid UI inconsistency issue",
			issue: SyncIssue{
				Type:        "ui_inconsistency",
				Description: "Button color doesn't match design system",
				Severity:    "warning",
				Suggestion:  "Apply correct CSS class",
				TestCase:    "should render button correctly",
				Location:    "button.spec.js:15",
			},
			wantValid: true,
		},
		{
			name: "invalid type",
			issue: SyncIssue{
				Type:        "invalid_type",
				Description: "Test description",
				Severity:    "critical",
				Suggestion:  "Test suggestion",
				TestCase:    "test case",
				Location:    "test.js:1",
			},
			wantValid: false,
			wantError: "type",
		},
		{
			name: "missing description",
			issue: SyncIssue{
				Type:        "api_mismatch",
				Description: "",
				Severity:    "critical",
				Suggestion:  "Test suggestion",
				TestCase:    "test case",
				Location:    "test.js:1",
			},
			wantValid: false,
			wantError: "description",
		},
		{
			name: "invalid severity",
			issue: SyncIssue{
				Type:        "api_mismatch",
				Description: "Test description",
				Severity:    "invalid_severity",
				Suggestion:  "Test suggestion",
				TestCase:    "test case",
				Location:    "test.js:1",
			},
			wantValid: false,
			wantError: "severity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.issue)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid issue, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid issue, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}
