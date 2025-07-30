package models

import (
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
)

func TestWSMessageValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		message   WSMessage
		wantValid bool
		wantError string
	}{
		{
			name: "valid sync status update message",
			message: WSMessage{
				Type:      "sync_status_update",
				Data:      map[string]interface{}{"status": "connected"},
				Timestamp: time.Now(),
				ClientID:  "client-123",
			},
			wantValid: true,
		},
		{
			name: "valid test progress message",
			message: WSMessage{
				Type:      "test_progress",
				Data:      map[string]interface{}{"progress": 50},
				Timestamp: time.Now(),
				ClientID:  "client-456",
			},
			wantValid: true,
		},
		{
			name: "invalid message type",
			message: WSMessage{
				Type:      "invalid_type",
				Data:      map[string]interface{}{},
				Timestamp: time.Now(),
				ClientID:  "client-123",
			},
			wantValid: false,
			wantError: "type",
		},
		{
			name: "missing client ID",
			message: WSMessage{
				Type:      "connect",
				Data:      map[string]interface{}{},
				Timestamp: time.Now(),
				ClientID:  "",
			},
			wantValid: false,
			wantError: "client_id",
		},
		{
			name: "missing type",
			message: WSMessage{
				Type:      "",
				Data:      map[string]interface{}{},
				Timestamp: time.Now(),
				ClientID:  "client-123",
			},
			wantValid: false,
			wantError: "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.message)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid message, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid message, but validation passed")
			}

			if !tt.wantValid && tt.wantError != "" {
				if _, exists := result.Errors[tt.wantError]; !exists {
					t.Errorf("Expected error for field %s, got errors: %v", tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestHealthStatusValidation(t *testing.T) {
	validator := utils.NewValidator()

	tests := []struct {
		name      string
		health    HealthStatus
		wantValid bool
	}{
		{
			name: "valid health status",
			health: HealthStatus{
				Frontend: true,
				Backend:  true,
				Database: false,
				Message:  "All systems operational",
			},
			wantValid: true,
		},
		{
			name: "valid health status with empty message",
			health: HealthStatus{
				Frontend: false,
				Backend:  true,
				Database: true,
				Message:  "",
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStruct(tt.health)

			if tt.wantValid && !result.IsValid {
				t.Errorf("Expected valid health status, got errors: %v", result.Errors)
			}

			if !tt.wantValid && result.IsValid {
				t.Errorf("Expected invalid health status, but validation passed")
			}
		})
	}
}
