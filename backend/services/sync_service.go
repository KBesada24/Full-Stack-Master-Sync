package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
)

// SyncService handles environment synchronization and connection management
type SyncService struct {
	environments map[string]*models.SyncEnvironment
	mutex        sync.RWMutex
	logger       *utils.Logger
	httpClient   *http.Client
	wsHub        WebSocketBroadcaster
}

// NewSyncService creates a new sync service instance
func NewSyncService(wsHub WebSocketBroadcaster) *SyncService {
	return &SyncService{
		environments: make(map[string]*models.SyncEnvironment),
		logger:       utils.GetLogger(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		wsHub: wsHub,
	}
}

// ConnectEnvironment establishes a connection to a sync environment
func (s *SyncService) ConnectEnvironment(req *models.SyncConnectionRequest) (*models.SyncStatusResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info("Attempting to connect to sync environment", map[string]interface{}{
		"environment":  req.Environment,
		"frontend_url": req.FrontendURL,
		"backend_url":  req.BackendURL,
	})

	// Validate URLs by making health check requests
	frontendHealthy, frontendErr := s.checkURLHealth(req.FrontendURL)
	backendHealthy, backendErr := s.checkURLHealth(req.BackendURL)

	// Create or update environment
	env := &models.SyncEnvironment{
		Name:        req.Environment,
		FrontendURL: req.FrontendURL,
		BackendURL:  req.BackendURL,
		LastChecked: time.Now(),
		Metadata:    make(map[string]string),
	}

	// Determine environment status
	if frontendHealthy && backendHealthy {
		env.Status = "active"
		env.Metadata["connection_status"] = "healthy"
	} else {
		env.Status = "error"
		if frontendErr != nil {
			env.Metadata["frontend_error"] = frontendErr.Error()
		}
		if backendErr != nil {
			env.Metadata["backend_error"] = backendErr.Error()
		}
	}

	// Store environment
	s.environments[req.Environment] = env

	// Create response
	response := &models.SyncStatusResponse{
		Status:    env.Status,
		Connected: env.Status == "active",
		LastSync:  env.LastChecked,
		Environments: map[string]string{
			req.Environment: env.Status,
		},
		Health: models.HealthStatus{
			Frontend: frontendHealthy,
			Backend:  backendHealthy,
			Database: true, // Assuming database is always healthy for now
			Message:  s.getHealthMessage(frontendHealthy, backendHealthy),
		},
	}

	s.logger.Info("Sync environment connection completed", map[string]interface{}{
		"environment": req.Environment,
		"status":      env.Status,
		"connected":   response.Connected,
	})

	// Broadcast sync status update via WebSocket
	s.broadcastSyncUpdate(response)

	return response, nil
}

// GetSyncStatus returns the current sync status for all environments
func (s *SyncService) GetSyncStatus() (*models.SyncStatusResponse, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	environments := make(map[string]string)
	overallStatus := "disconnected"
	connected := false
	lastSync := time.Time{}
	frontendHealthy := false
	backendHealthy := false

	// Aggregate status from all environments
	for name, env := range s.environments {
		environments[name] = env.Status

		if env.Status == "active" {
			connected = true
			overallStatus = "connected"
			frontendHealthy = true
			backendHealthy = true
		}

		if env.LastChecked.After(lastSync) {
			lastSync = env.LastChecked
		}
	}

	// If no environments, set default status
	if len(s.environments) == 0 {
		overallStatus = "disconnected"
	}

	response := &models.SyncStatusResponse{
		Status:       overallStatus,
		Connected:    connected,
		LastSync:     lastSync,
		Environments: environments,
		Health: models.HealthStatus{
			Frontend: frontendHealthy,
			Backend:  backendHealthy,
			Database: true,
			Message:  s.getHealthMessage(frontendHealthy, backendHealthy),
		},
	}

	s.logger.Debug("Retrieved sync status", map[string]interface{}{
		"status":       overallStatus,
		"connected":    connected,
		"environments": len(s.environments),
	})

	return response, nil
}

// ValidateEndpoint validates endpoint compatibility between frontend and backend
func (s *SyncService) ValidateEndpoint(req *models.SyncValidationRequest) (*models.SyncValidationResponse, error) {
	s.logger.Info("Validating endpoint compatibility", map[string]interface{}{
		"frontend_endpoint": req.FrontendEndpoint,
		"backend_endpoint":  req.BackendEndpoint,
		"method":            req.Method,
	})

	response := &models.SyncValidationResponse{
		IsCompatible: true,
		Issues:       []models.SyncCompatibilityIssue{},
		Suggestions:  []string{},
		ValidatedAt:  time.Now(),
	}

	// Validate frontend endpoint
	frontendResp, frontendErr := s.makeTestRequest(req.FrontendEndpoint, req.Method, req.Headers, req.Payload)

	// Validate backend endpoint
	backendResp, backendErr := s.makeTestRequest(req.BackendEndpoint, req.Method, req.Headers, req.Payload)

	// Compare responses and identify issues
	if frontendErr != nil {
		response.IsCompatible = false
		response.Issues = append(response.Issues, models.SyncCompatibilityIssue{
			Type:        "timeout",
			Field:       "frontend_endpoint",
			Expected:    "accessible",
			Actual:      "error",
			Severity:    "critical",
			Description: fmt.Sprintf("Frontend endpoint error: %v", frontendErr),
		})
		response.Suggestions = append(response.Suggestions, "Check if frontend server is running and accessible")
	}

	if backendErr != nil {
		response.IsCompatible = false
		response.Issues = append(response.Issues, models.SyncCompatibilityIssue{
			Type:        "timeout",
			Field:       "backend_endpoint",
			Expected:    "accessible",
			Actual:      "error",
			Severity:    "critical",
			Description: fmt.Sprintf("Backend endpoint error: %v", backendErr),
		})
		response.Suggestions = append(response.Suggestions, "Check if backend server is running and accessible")
	}

	// If both endpoints are accessible, compare responses
	if frontendErr == nil && backendErr == nil {
		s.compareResponses(frontendResp, backendResp, response)
	}

	s.logger.Info("Endpoint validation completed", map[string]interface{}{
		"is_compatible": response.IsCompatible,
		"issues_count":  len(response.Issues),
	})

	return response, nil
}

// checkURLHealth performs a health check on a given URL
func (s *SyncService) checkURLHealth(url string) (bool, error) {
	// Try to make a GET request to the URL
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to connect to %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx status codes as healthy
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, nil
	}

	return false, fmt.Errorf("unhealthy status code %d from %s", resp.StatusCode, url)
}

// makeTestRequest makes a test request to an endpoint
func (s *SyncService) makeTestRequest(url, method string, headers map[string]string, payload interface{}) (*http.Response, error) {
	var req *http.Request
	var err error

	// Create request based on method
	switch method {
	case "GET":
		req, err = http.NewRequest("GET", url, nil)
	case "POST", "PUT", "PATCH":
		var body []byte
		if payload != nil {
			body, err = json.Marshal(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal payload: %w", err)
			}
		}
		req, err = http.NewRequest(method, url, nil)
		if err == nil && body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	case "DELETE":
		req, err = http.NewRequest("DELETE", url, nil)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Make the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// compareResponses compares frontend and backend responses for compatibility
func (s *SyncService) compareResponses(frontendResp, backendResp *http.Response, response *models.SyncValidationResponse) {
	// Compare status codes
	if frontendResp.StatusCode != backendResp.StatusCode {
		response.IsCompatible = false
		response.Issues = append(response.Issues, models.SyncCompatibilityIssue{
			Type:        "status_code_mismatch",
			Field:       "status_code",
			Expected:    fmt.Sprintf("%d", frontendResp.StatusCode),
			Actual:      fmt.Sprintf("%d", backendResp.StatusCode),
			Severity:    "warning",
			Description: "Frontend and backend returned different status codes",
		})
		response.Suggestions = append(response.Suggestions, "Ensure both endpoints return consistent status codes")
	}

	// Compare content types
	frontendContentType := frontendResp.Header.Get("Content-Type")
	backendContentType := backendResp.Header.Get("Content-Type")

	if frontendContentType != backendContentType {
		response.Issues = append(response.Issues, models.SyncCompatibilityIssue{
			Type:        "header_mismatch",
			Field:       "content_type",
			Expected:    frontendContentType,
			Actual:      backendContentType,
			Severity:    "info",
			Description: "Content-Type headers differ between frontend and backend",
		})
		response.Suggestions = append(response.Suggestions, "Consider standardizing Content-Type headers")
	}

	// Close response bodies
	frontendResp.Body.Close()
	backendResp.Body.Close()
}

// getHealthMessage returns a descriptive health message
func (s *SyncService) getHealthMessage(frontendHealthy, backendHealthy bool) string {
	if frontendHealthy && backendHealthy {
		return "All services are healthy and connected"
	} else if !frontendHealthy && !backendHealthy {
		return "Both frontend and backend services are unreachable"
	} else if !frontendHealthy {
		return "Frontend service is unreachable"
	} else {
		return "Backend service is unreachable"
	}
}

// GetEnvironments returns all registered environments
func (s *SyncService) GetEnvironments() map[string]*models.SyncEnvironment {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Create a copy to avoid race conditions
	environments := make(map[string]*models.SyncEnvironment)
	for name, env := range s.environments {
		environments[name] = env
	}

	return environments
}

// RemoveEnvironment removes an environment from the sync service
func (s *SyncService) RemoveEnvironment(environmentName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.environments[environmentName]; !exists {
		return fmt.Errorf("environment '%s' not found", environmentName)
	}

	delete(s.environments, environmentName)

	s.logger.Info("Environment removed", map[string]interface{}{
		"environment": environmentName,
	})

	// Broadcast sync status update after environment removal
	if status, err := s.GetSyncStatus(); err == nil {
		s.broadcastSyncUpdate(status)
	}

	return nil
}

// broadcastSyncUpdate broadcasts sync status updates via WebSocket
func (s *SyncService) broadcastSyncUpdate(status *models.SyncStatusResponse) {
	if s.wsHub == nil {
		return
	}

	updateData := map[string]interface{}{
		"type":         "sync_status_change",
		"status":       status.Status,
		"connected":    status.Connected,
		"last_sync":    status.LastSync,
		"environments": status.Environments,
		"health":       status.Health,
		"timestamp":    time.Now(),
	}

	s.wsHub.BroadcastToAll("sync_status_update", updateData)

	s.logger.Debug("Broadcasted sync status update", map[string]interface{}{
		"status":    status.Status,
		"connected": status.Connected,
	})
}
