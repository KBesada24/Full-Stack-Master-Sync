package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSyncService is a mock implementation of the sync service
type MockSyncService struct {
	mock.Mock
}

func (m *MockSyncService) ConnectEnvironment(req *models.SyncConnectionRequest) (*models.SyncStatusResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SyncStatusResponse), args.Error(1)
}

func (m *MockSyncService) GetSyncStatus() (*models.SyncStatusResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SyncStatusResponse), args.Error(1)
}

func (m *MockSyncService) ValidateEndpoint(req *models.SyncValidationRequest) (*models.SyncValidationResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SyncValidationResponse), args.Error(1)
}

func (m *MockSyncService) GetEnvironments() map[string]*models.SyncEnvironment {
	args := m.Called()
	return args.Get(0).(map[string]*models.SyncEnvironment)
}

func (m *MockSyncService) RemoveEnvironment(environmentName string) error {
	args := m.Called(environmentName)
	return args.Error(0)
}

func setupTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})
	return app
}

func TestNewSyncHandler(t *testing.T) {
	mockService := &MockSyncService{}
	handler := NewSyncHandler(mockService)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.logger)
}

func TestSyncHandler_ConnectEnvironment(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockResponse   *models.SyncStatusResponse
		mockError      error
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful connection",
			requestBody: models.SyncConnectionRequest{
				Environment: "test",
				FrontendURL: "http://frontend.test",
				BackendURL:  "http://backend.test",
			},
			mockResponse: &models.SyncStatusResponse{
				Status:    "connected",
				Connected: true,
				LastSync:  time.Now(),
				Environments: map[string]string{
					"test": "active",
				},
				Health: models.HealthStatus{
					Frontend: true,
					Backend:  true,
					Database: true,
					Message:  "All services are healthy and connected",
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "invalid request body",
			requestBody: map[string]interface{}{
				"invalid": "data",
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "service error",
			requestBody: models.SyncConnectionRequest{
				Environment: "test",
				FrontendURL: "http://frontend.test",
				BackendURL:  "http://backend.test",
			},
			mockResponse:   nil,
			mockError:      errors.New("connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name: "validation error - missing environment",
			requestBody: models.SyncConnectionRequest{
				FrontendURL: "http://frontend.test",
				BackendURL:  "http://backend.test",
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "validation error - invalid URL",
			requestBody: models.SyncConnectionRequest{
				Environment: "test",
				FrontendURL: "invalid-url",
				BackendURL:  "http://backend.test",
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			app := setupTestApp()
			mockService := &MockSyncService{}
			handler := NewSyncHandler(mockService)

			// Setup mock expectations
			if !tt.expectedError || tt.mockError != nil {
				mockService.On("ConnectEnvironment", mock.AnythingOfType("*models.SyncConnectionRequest")).Return(tt.mockResponse, tt.mockError)
			}

			// Setup route
			app.Post("/api/sync/connect", handler.ConnectEnvironment)

			// Create request
			requestBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/sync/connect", bytes.NewReader(requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verify mock expectations
			if !tt.expectedError || tt.mockError != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestSyncHandler_GetSyncStatus(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   *models.SyncStatusResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful status retrieval",
			mockResponse: &models.SyncStatusResponse{
				Status:    "connected",
				Connected: true,
				LastSync:  time.Now(),
				Environments: map[string]string{
					"test": "active",
				},
				Health: models.HealthStatus{
					Frontend: true,
					Backend:  true,
					Database: true,
					Message:  "All services are healthy and connected",
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "service error",
			mockResponse:   nil,
			mockError:      errors.New("status retrieval failed"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			app := setupTestApp()
			mockService := &MockSyncService{}
			handler := NewSyncHandler(mockService)

			// Setup mock expectations
			mockService.On("GetSyncStatus").Return(tt.mockResponse, tt.mockError)

			// Setup route
			app.Get("/api/sync/status", handler.GetSyncStatus)

			// Create request
			req := httptest.NewRequest("GET", "/api/sync/status", nil)

			// Execute request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}

func TestSyncHandler_ValidateEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockResponse   *models.SyncValidationResponse
		mockError      error
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful validation",
			requestBody: models.SyncValidationRequest{
				FrontendEndpoint: "http://frontend.test/api/test",
				BackendEndpoint:  "http://backend.test/api/test",
				Method:           "GET",
			},
			mockResponse: &models.SyncValidationResponse{
				IsCompatible: true,
				Issues:       []models.SyncCompatibilityIssue{},
				Suggestions:  []string{},
				ValidatedAt:  time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "validation with issues",
			requestBody: models.SyncValidationRequest{
				FrontendEndpoint: "http://frontend.test/api/test",
				BackendEndpoint:  "http://backend.test/api/test",
				Method:           "GET",
			},
			mockResponse: &models.SyncValidationResponse{
				IsCompatible: false,
				Issues: []models.SyncCompatibilityIssue{
					{
						Type:        "status_code_mismatch",
						Field:       "status_code",
						Expected:    "200",
						Actual:      "500",
						Severity:    "critical",
						Description: "Status codes don't match",
					},
				},
				Suggestions: []string{"Check backend implementation"},
				ValidatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "invalid request body",
			requestBody: map[string]interface{}{
				"invalid": "data",
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "service error",
			requestBody: models.SyncValidationRequest{
				FrontendEndpoint: "http://frontend.test/api/test",
				BackendEndpoint:  "http://backend.test/api/test",
				Method:           "GET",
			},
			mockResponse:   nil,
			mockError:      errors.New("validation failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name: "validation error - missing endpoint",
			requestBody: models.SyncValidationRequest{
				BackendEndpoint: "http://backend.test/api/test",
				Method:          "GET",
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "validation error - invalid method",
			requestBody: models.SyncValidationRequest{
				FrontendEndpoint: "http://frontend.test/api/test",
				BackendEndpoint:  "http://backend.test/api/test",
				Method:           "INVALID",
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			app := setupTestApp()
			mockService := &MockSyncService{}
			handler := NewSyncHandler(mockService)

			// Setup mock expectations
			if !tt.expectedError || tt.mockError != nil {
				mockService.On("ValidateEndpoint", mock.AnythingOfType("*models.SyncValidationRequest")).Return(tt.mockResponse, tt.mockError)
			}

			// Setup route
			app.Post("/api/sync/validate", handler.ValidateEndpoint)

			// Create request
			requestBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/sync/validate", bytes.NewReader(requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verify mock expectations
			if !tt.expectedError || tt.mockError != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestSyncHandler_GetEnvironments(t *testing.T) {
	tests := []struct {
		name             string
		mockEnvironments map[string]*models.SyncEnvironment
		expectedStatus   int
	}{
		{
			name:             "empty environments",
			mockEnvironments: map[string]*models.SyncEnvironment{},
			expectedStatus:   http.StatusOK,
		},
		{
			name: "with environments",
			mockEnvironments: map[string]*models.SyncEnvironment{
				"test1": {
					Name:        "test1",
					FrontendURL: "http://frontend1.test",
					BackendURL:  "http://backend1.test",
					Status:      "active",
					LastChecked: time.Now(),
				},
				"test2": {
					Name:        "test2",
					FrontendURL: "http://frontend2.test",
					BackendURL:  "http://backend2.test",
					Status:      "error",
					LastChecked: time.Now(),
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			app := setupTestApp()
			mockService := &MockSyncService{}
			handler := NewSyncHandler(mockService)

			// Setup mock expectations
			mockService.On("GetEnvironments").Return(tt.mockEnvironments)

			// Setup route
			app.Get("/api/sync/environments", handler.GetEnvironments)

			// Create request
			req := httptest.NewRequest("GET", "/api/sync/environments", nil)

			// Execute request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}

func TestSyncHandler_RemoveEnvironment(t *testing.T) {
	tests := []struct {
		name            string
		environmentName string
		mockError       error
		expectedStatus  int
	}{
		{
			name:            "successful removal",
			environmentName: "test",
			mockError:       nil,
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "environment not found",
			environmentName: "nonexistent",
			mockError:       errors.New("environment 'nonexistent' not found"),
			expectedStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			app := setupTestApp()
			mockService := &MockSyncService{}
			handler := NewSyncHandler(mockService)

			// Setup mock expectations
			if tt.environmentName != "" {
				mockService.On("RemoveEnvironment", tt.environmentName).Return(tt.mockError)
			}

			// Setup route
			app.Delete("/api/sync/environments/:name", handler.RemoveEnvironment)

			// Create request
			var url string
			if tt.environmentName == "" {
				url = "/api/sync/environments/"
			} else {
				url = "/api/sync/environments/" + tt.environmentName
			}
			req := httptest.NewRequest("DELETE", url, nil)

			// Execute request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verify mock expectations
			if tt.environmentName != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestSyncHandler_Integration(t *testing.T) {
	// This test uses the real sync service to test the full integration
	app := setupTestApp()
	syncService := services.NewSyncService()
	handler := NewSyncHandler(syncService)

	// Setup routes
	app.Post("/api/sync/connect", handler.ConnectEnvironment)
	app.Get("/api/sync/status", handler.GetSyncStatus)
	app.Get("/api/sync/environments", handler.GetEnvironments)
	app.Delete("/api/sync/environments/:name", handler.RemoveEnvironment)

	t.Run("full workflow", func(t *testing.T) {
		// 1. Check initial status (should be disconnected)
		req := httptest.NewRequest("GET", "/api/sync/status", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 2. Get environments (should be empty)
		req = httptest.NewRequest("GET", "/api/sync/environments", nil)
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 3. Try to remove non-existent environment
		req = httptest.NewRequest("DELETE", "/api/sync/environments/nonexistent", nil)
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
