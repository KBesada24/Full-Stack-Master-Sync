package models

import "time"

// SyncConnectionRequest represents a request to establish sync connection
type SyncConnectionRequest struct {
	FrontendURL string `json:"frontend_url" validate:"required,url"`
	BackendURL  string `json:"backend_url" validate:"required,url"`
	Environment string `json:"environment" validate:"required,min=1,max=50"`
}

// SyncStatusResponse represents the current sync status
type SyncStatusResponse struct {
	Status       string            `json:"status" validate:"required,oneof=connected disconnected connecting error"`
	Connected    bool              `json:"connected"`
	LastSync     time.Time         `json:"last_sync"`
	Environments map[string]string `json:"environments"`
	Health       HealthStatus      `json:"health"`
}

// SyncValidationRequest represents a request to validate endpoint compatibility
type SyncValidationRequest struct {
	FrontendEndpoint string            `json:"frontend_endpoint" validate:"required,url"`
	BackendEndpoint  string            `json:"backend_endpoint" validate:"required,url"`
	Method           string            `json:"method" validate:"required,oneof=GET POST PUT DELETE PATCH"`
	Headers          map[string]string `json:"headers"`
	Payload          interface{}       `json:"payload"`
}

// SyncValidationResponse represents the result of endpoint validation
type SyncValidationResponse struct {
	IsCompatible bool                     `json:"is_compatible"`
	Issues       []SyncCompatibilityIssue `json:"issues,omitempty"`
	Suggestions  []string                 `json:"suggestions,omitempty"`
	ValidatedAt  time.Time                `json:"validated_at"`
}

// SyncCompatibilityIssue represents a compatibility issue found during validation
type SyncCompatibilityIssue struct {
	Type        string `json:"type" validate:"required,oneof=schema_mismatch status_code_mismatch header_mismatch timeout"`
	Field       string `json:"field,omitempty"`
	Expected    string `json:"expected,omitempty"`
	Actual      string `json:"actual,omitempty"`
	Severity    string `json:"severity" validate:"required,oneof=critical warning info"`
	Description string `json:"description"`
}

// SyncEnvironment represents a sync environment configuration
type SyncEnvironment struct {
	Name        string            `json:"name" validate:"required,min=1,max=50"`
	FrontendURL string            `json:"frontend_url" validate:"required,url"`
	BackendURL  string            `json:"backend_url" validate:"required,url"`
	Status      string            `json:"status" validate:"required,oneof=active inactive error"`
	LastChecked time.Time         `json:"last_checked"`
	Metadata    map[string]string `json:"metadata"`
}
