package models

import (
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
)

func TestSyncConnectionRequestValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		request   SyncConnectionRequest
		wantValid bool
		wantError string
	}{
		{
			name: "valid sync connection request",
			request: SyncConnectionRequest{
				FrontendURL: "http://localhost:3000",
				BackendURL:  "http://localhost:8080",
				Environment: "development",
			},
			wantValid: true,
		},
		{
			name: "valid sync connection request with HTTPS",
			request: SyncConnectionRequest{
				FrontendURL: "https://app.example.com",
				BackendURL:  "https://api.example.com",
				Environment: "production",
			},
			wantValid: true,
		},
		{
			name: "missing frontend URL",
			request: SyncConnectionRequest{
				FrontendURL: "",
				BackendURL:  "http://localhost:8080",
				Environment: "development",
			},
			wantValid: false,
			wantError: "frontend_url",
		},
		{
			name: "invalid frontend URL",
			request: SyncConnectionRequest{
				FrontendURL: "not-a-url",
				BackendURL:  "http://localhost:8080",
				Environment: "development",
			},
			wantValid: false,
			wantError: "frontend_url",
		},
		{
			name: "missing backend URL",
			request: SyncConnectionRequest{
				FrontendURL: "http://localhost:3000",
				BackendURL:  "",
				Environment: "development",
			},
			wantValid: false,
			wantError: "backend_url",
		},
		{
			name: "invalid backend URL",
			request: SyncConnectionRequest{
				FrontendURL: "http://localhost:3000",
				BackendURL:  "invalid-url",
				Environment: "development",
			},
			wantValid: false,
			wantError: "backend_url",
		},
		{
			name: "missing environment",
			request: SyncConnectionRequest{
				FrontendURL: "http://localhost:3000",
				BackendURL:  "http://localhost:8080",
				Environment: "",
			},
			wantValid: false,
			wantError: "environment",
		},
		{
			name: "environment too long",
			request: SyncConnectionRequest{
				FrontendURL: "http://localhost:3000",
				BackendURL:  "http://localhost:8080",
				Environment: "this-is-a-very-long-environment-name-that-exceeds-the-maximum-allowed-length",
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

func TestSyncStatusResponseValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		response  SyncStatusResponse
		wantValid bool
		wantError string
	}{
		{
			name: "valid connected status",
			response: SyncStatusResponse{
				Status:    "connected",
				Connected: true,
				LastSync:  time.Now(),
				Environments: map[string]string{
					"frontend": "http://localhost:3000",
					"backend":  "http://localhost:8080",
				},
				Health: HealthStatus{
					Frontend: true,
					Backend:  true,
					Database: true,
					Message:  "All systems operational",
				},
			},
			wantValid: true,
		},
		{
			name: "valid disconnected status",
			response: SyncStatusResponse{
				Status:       "disconnected",
				Connected:    false,
				LastSync:     time.Now().Add(-time.Hour),
				Environments: map[string]string{},
				Health: HealthStatus{
					Frontend: false,
					Backend:  false,
					Database: true,
					Message:  "Connection lost",
				},
			},
			wantValid: true,
		},
		{
			name: "invalid status",
			response: SyncStatusResponse{
				Status:       "invalid_status",
				Connected:    true,
				LastSync:     time.Now(),
				Environments: map[string]string{},
				Health:       HealthStatus{},
			},
			wantValid: false,
			wantError: "status",
		},
		{
			name: "missing status",
			response: SyncStatusResponse{
				Status:       "",
				Connected:    true,
				LastSync:     time.Now(),
				Environments: map[string]string{},
				Health:       HealthStatus{},
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

func TestSyncValidationRequestValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		request   SyncValidationRequest
		wantValid bool
		wantError string
	}{
		{
			name: "valid GET request",
			request: SyncValidationRequest{
				FrontendEndpoint: "http://localhost:3000/api/users",
				BackendEndpoint:  "http://localhost:8080/api/users",
				Method:           "GET",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Payload: nil,
			},
			wantValid: true,
		},
		{
			name: "valid POST request with payload",
			request: SyncValidationRequest{
				FrontendEndpoint: "http://localhost:3000/api/users",
				BackendEndpoint:  "http://localhost:8080/api/users",
				Method:           "POST",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Payload: map[string]interface{}{
					"name":  "John Doe",
					"email": "john@example.com",
				},
			},
			wantValid: true,
		},
		{
			name: "invalid method",
			request: SyncValidationRequest{
				FrontendEndpoint: "http://localhost:3000/api/users",
				BackendEndpoint:  "http://localhost:8080/api/users",
				Method:           "INVALID",
				Headers:          map[string]string{},
				Payload:          nil,
			},
			wantValid: false,
			wantError: "method",
		},
		{
			name: "missing frontend endpoint",
			request: SyncValidationRequest{
				FrontendEndpoint: "",
				BackendEndpoint:  "http://localhost:8080/api/users",
				Method:           "GET",
				Headers:          map[string]string{},
				Payload:          nil,
			},
			wantValid: false,
			wantError: "frontend_endpoint",
		},
		{
			name: "invalid backend endpoint",
			request: SyncValidationRequest{
				FrontendEndpoint: "http://localhost:3000/api/users",
				BackendEndpoint:  "not-a-url",
				Method:           "GET",
				Headers:          map[string]string{},
				Payload:          nil,
			},
			wantValid: false,
			wantError: "backend_endpoint",
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

func TestSyncCompatibilityIssueValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		issue     SyncCompatibilityIssue
		wantValid bool
		wantError string
	}{
		{
			name: "valid schema mismatch issue",
			issue: SyncCompatibilityIssue{
				Type:        "schema_mismatch",
				Field:       "user.email",
				Expected:    "string",
				Actual:      "number",
				Severity:    "critical",
				Description: "Email field type mismatch",
			},
			wantValid: true,
		},
		{
			name: "valid timeout issue",
			issue: SyncCompatibilityIssue{
				Type:        "timeout",
				Field:       "",
				Expected:    "5s",
				Actual:      "30s",
				Severity:    "warning",
				Description: "Response timeout exceeds expected threshold",
			},
			wantValid: true,
		},
		{
			name: "invalid type",
			issue: SyncCompatibilityIssue{
				Type:        "invalid_type",
				Field:       "test",
				Expected:    "value",
				Actual:      "other",
				Severity:    "critical",
				Description: "Test issue",
			},
			wantValid: false,
			wantError: "type",
		},
		{
			name: "invalid severity",
			issue: SyncCompatibilityIssue{
				Type:        "schema_mismatch",
				Field:       "test",
				Expected:    "value",
				Actual:      "other",
				Severity:    "invalid_severity",
				Description: "Test issue",
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
