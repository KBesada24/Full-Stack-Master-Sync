package models

import (
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
)

func TestLogEntryValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		logEntry  LogEntry
		wantValid bool
		wantError string
	}{
		{
			name: "valid error log entry",
			logEntry: LogEntry{
				ID:        "log-123",
				Timestamp: time.Now(),
				Level:     "error",
				Source:    "frontend",
				Message:   "Failed to fetch user data",
				Context: map[string]interface{}{
					"userId": "user-456",
					"url":    "/api/users/456",
				},
				StackTrace: "Error at line 15 in user.service.js",
				UserID:     "user-456",
				SessionID:  "session-789",
				Component:  "UserService",
				Function:   "fetchUser",
				LineNumber: 15,
			},
			wantValid: true,
		},
		{
			name: "valid info log entry",
			logEntry: LogEntry{
				ID:        "log-456",
				Timestamp: time.Now(),
				Level:     "info",
				Source:    "backend",
				Message:   "User logged in successfully",
				Context: map[string]interface{}{
					"userId": "user-123",
					"ip":     "192.168.1.1",
				},
				UserID:    "user-123",
				SessionID: "session-456",
				Component: "AuthController",
				Function:  "login",
			},
			wantValid: true,
		},
		{
			name: "missing ID",
			logEntry: LogEntry{
				ID:        "",
				Timestamp: time.Now(),
				Level:     "error",
				Source:    "frontend",
				Message:   "Test message",
			},
			wantValid: false,
			wantError: "id",
		},
		{
			name: "invalid level",
			logEntry: LogEntry{
				ID:        "log-789",
				Timestamp: time.Now(),
				Level:     "invalid_level",
				Source:    "frontend",
				Message:   "Test message",
			},
			wantValid: false,
			wantError: "level",
		},
		{
			name: "invalid source",
			logEntry: LogEntry{
				ID:        "log-999",
				Timestamp: time.Now(),
				Level:     "info",
				Source:    "invalid_source",
				Message:   "Test message",
			},
			wantValid: false,
			wantError: "source",
		},
		{
			name: "missing message",
			logEntry: LogEntry{
				ID:        "log-111",
				Timestamp: time.Now(),
				Level:     "info",
				Source:    "frontend",
				Message:   "",
			},
			wantValid: false,
			wantError: "message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.logEntry)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid log entry, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid log entry, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestLogSubmissionRequestValidation(t *testing.T) {
	validator := utils.NewValidator()

	validLogEntry := LogEntry{
		ID:        "log-123",
		Timestamp: time.Now(),
		Level:     "info",
		Source:    "frontend",
		Message:   "Test message",
	}

	tests := []struct {
		name      string
		request   LogSubmissionRequest
		wantValid bool
		wantError string
	}{
		{
			name: "valid log submission request",
			request: LogSubmissionRequest{
				Logs:    []LogEntry{validLogEntry},
				BatchID: "batch-123",
				Source:  "frontend",
				Metadata: map[string]string{
					"version": "1.0.0",
					"env":     "production",
				},
			},
			wantValid: true,
		},
		{
			name: "valid request with multiple logs",
			request: LogSubmissionRequest{
				Logs: []LogEntry{
					validLogEntry,
					{
						ID:        "log-456",
						Timestamp: time.Now(),
						Level:     "error",
						Source:    "frontend",
						Message:   "Another test message",
					},
				},
				BatchID:  "batch-456",
				Source:   "frontend",
				Metadata: map[string]string{},
			},
			wantValid: true,
		},
		{
			name: "empty logs array",
			request: LogSubmissionRequest{
				Logs:     []LogEntry{},
				BatchID:  "batch-789",
				Source:   "frontend",
				Metadata: map[string]string{},
			},
			wantValid: false,
			wantError: "logs",
		},
		{
			name: "invalid source",
			request: LogSubmissionRequest{
				Logs:     []LogEntry{validLogEntry},
				BatchID:  "batch-999",
				Source:   "invalid_source",
				Metadata: map[string]string{},
			},
			wantValid: false,
			wantError: "source",
		},
		{
			name: "missing source",
			request: LogSubmissionRequest{
				Logs:     []LogEntry{validLogEntry},
				BatchID:  "batch-111",
				Source:   "",
				Metadata: map[string]string{},
			},
			wantValid: false,
			wantError: "source",
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

func TestLogAnalysisRequestValidation(t *testing.T) {
	validator := utils.NewValidator()

	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)

	tests := []struct {
		name      string
		request   LogAnalysisRequest
		wantValid bool
		wantError string
	}{
		{
			name: "valid log analysis request",
			request: LogAnalysisRequest{
				TimeRange: TimeRange{
					Start: oneHourAgo,
					End:   now,
				},
				Levels:      []string{"error", "warn"},
				Sources:     []string{"frontend", "backend"},
				Components:  []string{"UserService", "AuthController"},
				SearchQuery: "failed to fetch",
				Filters: map[string]string{
					"userId": "user-123",
				},
				Limit: 100,
			},
			wantValid: true,
		},
		{
			name: "valid request with minimal fields",
			request: LogAnalysisRequest{
				TimeRange: TimeRange{
					Start: oneHourAgo,
					End:   now,
				},
				Levels:      []string{},
				Sources:     []string{},
				Components:  []string{},
				SearchQuery: "",
				Filters:     map[string]string{},
				Limit:       50,
			},
			wantValid: true,
		},
		{
			name: "valid request with mixed levels",
			request: LogAnalysisRequest{
				TimeRange: TimeRange{
					Start: oneHourAgo,
					End:   now,
				},
				Levels:  []string{"error", "warn"},
				Sources: []string{"frontend"},
				Limit:   100,
			},
			wantValid: true,
		},
		{
			name: "valid request with mixed sources",
			request: LogAnalysisRequest{
				TimeRange: TimeRange{
					Start: oneHourAgo,
					End:   now,
				},
				Levels:  []string{"error"},
				Sources: []string{"frontend", "backend"},
				Limit:   100,
			},
			wantValid: true,
		},
		{
			name: "limit too low",
			request: LogAnalysisRequest{
				TimeRange: TimeRange{
					Start: oneHourAgo,
					End:   now,
				},
				Levels:  []string{"error"},
				Sources: []string{"frontend"},
				Limit:   0,
			},
			wantValid: false,
			wantError: "limit",
		},
		{
			name: "limit too high",
			request: LogAnalysisRequest{
				TimeRange: TimeRange{
					Start: oneHourAgo,
					End:   now,
				},
				Levels:  []string{"error"},
				Sources: []string{"frontend"},
				Limit:   1001,
			},
			wantValid: false,
			wantError: "limit",
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

func TestLogIssueValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		issue     LogIssue
		wantValid bool
		wantError string
	}{
		{
			name: "valid error spike issue",
			issue: LogIssue{
				Type:               "error_spike",
				Count:              25,
				FirstSeen:          time.Now().Add(-time.Hour),
				LastSeen:           time.Now(),
				Description:        "Sudden increase in authentication errors",
				Severity:           "critical",
				Solution:           "Check authentication service health",
				AffectedComponents: []string{"AuthService", "LoginController"},
				SampleLogs: []LogEntry{
					{
						ID:        "log-123",
						Timestamp: time.Now(),
						Level:     "error",
						Source:    "backend",
						Message:   "Authentication failed",
					},
				},
			},
			wantValid: true,
		},
		{
			name: "valid performance issue",
			issue: LogIssue{
				Type:               "performance_degradation",
				Count:              10,
				FirstSeen:          time.Now().Add(-30 * time.Minute),
				LastSeen:           time.Now(),
				Description:        "API response times increased significantly",
				Severity:           "high",
				Solution:           "Investigate database performance",
				AffectedComponents: []string{"UserAPI", "DatabaseService"},
				SampleLogs:         []LogEntry{},
			},
			wantValid: true,
		},
		{
			name: "invalid type",
			issue: LogIssue{
				Type:               "invalid_type",
				Count:              5,
				FirstSeen:          time.Now().Add(-time.Hour),
				LastSeen:           time.Now(),
				Description:        "Test issue",
				Severity:           "medium",
				Solution:           "Test solution",
				AffectedComponents: []string{},
				SampleLogs:         []LogEntry{},
			},
			wantValid: false,
			wantError: "type",
		},
		{
			name: "count too low",
			issue: LogIssue{
				Type:               "error_spike",
				Count:              0,
				FirstSeen:          time.Now().Add(-time.Hour),
				LastSeen:           time.Now(),
				Description:        "Test issue",
				Severity:           "medium",
				Solution:           "Test solution",
				AffectedComponents: []string{},
				SampleLogs:         []LogEntry{},
			},
			wantValid: false,
			wantError: "count",
		},
		{
			name: "missing description",
			issue: LogIssue{
				Type:               "error_spike",
				Count:              5,
				FirstSeen:          time.Now().Add(-time.Hour),
				LastSeen:           time.Now(),
				Description:        "",
				Severity:           "medium",
				Solution:           "Test solution",
				AffectedComponents: []string{},
				SampleLogs:         []LogEntry{},
			},
			wantValid: false,
			wantError: "description",
		},
		{
			name: "invalid severity",
			issue: LogIssue{
				Type:               "error_spike",
				Count:              5,
				FirstSeen:          time.Now().Add(-time.Hour),
				LastSeen:           time.Now(),
				Description:        "Test issue",
				Severity:           "invalid_severity",
				Solution:           "Test solution",
				AffectedComponents: []string{},
				SampleLogs:         []LogEntry{},
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
