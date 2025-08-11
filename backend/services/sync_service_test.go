package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSyncService(t *testing.T) {
	service := NewSyncService(nil)

	assert.NotNil(t, service)
	assert.NotNil(t, service.environments)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.httpClient)
	assert.Equal(t, 10*time.Second, service.httpClient.Timeout)
}

func TestSyncService_ConnectEnvironment(t *testing.T) {
	tests := []struct {
		name           string
		request        *models.SyncConnectionRequest
		frontendStatus int
		backendStatus  int
		expectedStatus string
		expectedError  bool
	}{
		{
			name: "successful connection",
			request: &models.SyncConnectionRequest{
				Environment: "test",
				FrontendURL: "http://frontend.test",
				BackendURL:  "http://backend.test",
			},
			frontendStatus: http.StatusOK,
			backendStatus:  http.StatusOK,
			expectedStatus: "active",
			expectedError:  false,
		},
		{
			name: "frontend unreachable",
			request: &models.SyncConnectionRequest{
				Environment: "test",
				FrontendURL: "http://frontend.test",
				BackendURL:  "http://backend.test",
			},
			frontendStatus: http.StatusInternalServerError,
			backendStatus:  http.StatusOK,
			expectedStatus: "error",
			expectedError:  false,
		},
		{
			name: "backend unreachable",
			request: &models.SyncConnectionRequest{
				Environment: "test",
				FrontendURL: "http://frontend.test",
				BackendURL:  "http://backend.test",
			},
			frontendStatus: http.StatusOK,
			backendStatus:  http.StatusInternalServerError,
			expectedStatus: "error",
			expectedError:  false,
		},
		{
			name: "both unreachable",
			request: &models.SyncConnectionRequest{
				Environment: "test",
				FrontendURL: "http://frontend.test",
				BackendURL:  "http://backend.test",
			},
			frontendStatus: http.StatusInternalServerError,
			backendStatus:  http.StatusInternalServerError,
			expectedStatus: "error",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock servers
			frontendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.frontendStatus)
			}))
			defer frontendServer.Close()

			backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.backendStatus)
			}))
			defer backendServer.Close()

			// Update request URLs to use mock servers
			tt.request.FrontendURL = frontendServer.URL
			tt.request.BackendURL = backendServer.URL

			// Create service and test
			service := NewSyncService(nil)
			response, err := service.ConnectEnvironment(tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expectedStatus, response.Status)
			assert.Equal(t, tt.expectedStatus == "active", response.Connected)
			assert.Contains(t, response.Environments, tt.request.Environment)
			assert.Equal(t, tt.expectedStatus, response.Environments[tt.request.Environment])

			// Verify environment was stored
			environments := service.GetEnvironments()
			assert.Contains(t, environments, tt.request.Environment)
			assert.Equal(t, tt.expectedStatus, environments[tt.request.Environment].Status)
		})
	}
}

func TestSyncService_GetSyncStatus(t *testing.T) {
	service := NewSyncService(nil)

	t.Run("no environments", func(t *testing.T) {
		response, err := service.GetSyncStatus()

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "disconnected", response.Status)
		assert.False(t, response.Connected)
		assert.Empty(t, response.Environments)
	})

	t.Run("with active environment", func(t *testing.T) {
		// Add a mock environment
		service.environments["test"] = &models.SyncEnvironment{
			Name:        "test",
			FrontendURL: "http://frontend.test",
			BackendURL:  "http://backend.test",
			Status:      "active",
			LastChecked: time.Now(),
		}

		response, err := service.GetSyncStatus()

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "connected", response.Status)
		assert.True(t, response.Connected)
		assert.Contains(t, response.Environments, "test")
		assert.Equal(t, "active", response.Environments["test"])
		assert.True(t, response.Health.Frontend)
		assert.True(t, response.Health.Backend)
	})

	t.Run("with error environment", func(t *testing.T) {
		// Clear environments and add error environment
		service.environments = make(map[string]*models.SyncEnvironment)
		service.environments["test"] = &models.SyncEnvironment{
			Name:        "test",
			FrontendURL: "http://frontend.test",
			BackendURL:  "http://backend.test",
			Status:      "error",
			LastChecked: time.Now(),
		}

		response, err := service.GetSyncStatus()

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "disconnected", response.Status)
		assert.False(t, response.Connected)
		assert.Contains(t, response.Environments, "test")
		assert.Equal(t, "error", response.Environments["test"])
	})
}

func TestSyncService_ValidateEndpoint(t *testing.T) {
	tests := []struct {
		name                string
		request             *models.SyncValidationRequest
		frontendResponse    map[string]interface{}
		backendResponse     map[string]interface{}
		frontendStatus      int
		backendStatus       int
		expectedCompatible  bool
		expectedIssuesCount int
	}{
		{
			name: "compatible endpoints",
			request: &models.SyncValidationRequest{
				FrontendEndpoint: "http://frontend.test/api/test",
				BackendEndpoint:  "http://backend.test/api/test",
				Method:           "GET",
			},
			frontendResponse:    map[string]interface{}{"status": "ok"},
			backendResponse:     map[string]interface{}{"status": "ok"},
			frontendStatus:      http.StatusOK,
			backendStatus:       http.StatusOK,
			expectedCompatible:  true,
			expectedIssuesCount: 0,
		},
		{
			name: "status code mismatch",
			request: &models.SyncValidationRequest{
				FrontendEndpoint: "http://frontend.test/api/test",
				BackendEndpoint:  "http://backend.test/api/test",
				Method:           "GET",
			},
			frontendResponse:    map[string]interface{}{"status": "ok"},
			backendResponse:     map[string]interface{}{"status": "error"},
			frontendStatus:      http.StatusOK,
			backendStatus:       http.StatusInternalServerError,
			expectedCompatible:  false,
			expectedIssuesCount: 1,
		},
		{
			name: "frontend unreachable",
			request: &models.SyncValidationRequest{
				FrontendEndpoint: "http://invalid.test/api/test",
				BackendEndpoint:  "http://backend.test/api/test",
				Method:           "GET",
			},
			backendResponse:     map[string]interface{}{"status": "ok"},
			backendStatus:       http.StatusOK,
			expectedCompatible:  false,
			expectedIssuesCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var frontendServer *httptest.Server
			var backendServer *httptest.Server

			// Create frontend server if not testing unreachable scenario
			if tt.name != "frontend unreachable" {
				frontendServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.frontendStatus)
					json.NewEncoder(w).Encode(tt.frontendResponse)
				}))
				defer frontendServer.Close()
				tt.request.FrontendEndpoint = frontendServer.URL + "/api/test"
			}

			// Create backend server
			backendServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.backendStatus)
				json.NewEncoder(w).Encode(tt.backendResponse)
			}))
			defer backendServer.Close()
			tt.request.BackendEndpoint = backendServer.URL + "/api/test"

			// Create service and test
			service := NewSyncService(nil)
			response, err := service.ValidateEndpoint(tt.request)

			require.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expectedCompatible, response.IsCompatible)
			assert.Len(t, response.Issues, tt.expectedIssuesCount)
			assert.NotZero(t, response.ValidatedAt)

			// Check issue types if any
			if tt.expectedIssuesCount > 0 {
				for _, issue := range response.Issues {
					assert.NotEmpty(t, issue.Type)
					assert.NotEmpty(t, issue.Severity)
					assert.NotEmpty(t, issue.Description)
				}
			}
		})
	}
}

func TestSyncService_ValidateEndpoint_Methods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			// Create mock servers
			frontendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, method, r.Method)
				w.WriteHeader(http.StatusOK)
			}))
			defer frontendServer.Close()

			backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, method, r.Method)
				w.WriteHeader(http.StatusOK)
			}))
			defer backendServer.Close()

			request := &models.SyncValidationRequest{
				FrontendEndpoint: frontendServer.URL,
				BackendEndpoint:  backendServer.URL,
				Method:           method,
				Headers:          map[string]string{"X-Test": "value"},
				Payload:          map[string]interface{}{"test": "data"},
			}

			service := NewSyncService(nil)
			response, err := service.ValidateEndpoint(request)

			require.NoError(t, err)
			assert.NotNil(t, response)
		})
	}
}

func TestSyncService_ValidateEndpoint_UnsupportedMethod(t *testing.T) {
	service := NewSyncService(nil)

	request := &models.SyncValidationRequest{
		FrontendEndpoint: "http://frontend.test",
		BackendEndpoint:  "http://backend.test",
		Method:           "INVALID",
	}

	response, err := service.ValidateEndpoint(request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.False(t, response.IsCompatible)
	assert.Greater(t, len(response.Issues), 0)
}

func TestSyncService_GetEnvironments(t *testing.T) {
	service := NewSyncService(nil)

	t.Run("empty environments", func(t *testing.T) {
		environments := service.GetEnvironments()
		assert.Empty(t, environments)
	})

	t.Run("with environments", func(t *testing.T) {
		// Add test environments
		testEnv1 := &models.SyncEnvironment{
			Name:        "test1",
			FrontendURL: "http://frontend1.test",
			BackendURL:  "http://backend1.test",
			Status:      "active",
			LastChecked: time.Now(),
		}
		testEnv2 := &models.SyncEnvironment{
			Name:        "test2",
			FrontendURL: "http://frontend2.test",
			BackendURL:  "http://backend2.test",
			Status:      "error",
			LastChecked: time.Now(),
		}

		service.environments["test1"] = testEnv1
		service.environments["test2"] = testEnv2

		environments := service.GetEnvironments()

		assert.Len(t, environments, 2)
		assert.Contains(t, environments, "test1")
		assert.Contains(t, environments, "test2")
		assert.Equal(t, "active", environments["test1"].Status)
		assert.Equal(t, "error", environments["test2"].Status)
	})
}

func TestSyncService_RemoveEnvironment(t *testing.T) {
	service := NewSyncService(nil)

	t.Run("remove existing environment", func(t *testing.T) {
		// Add test environment
		service.environments["test"] = &models.SyncEnvironment{
			Name:   "test",
			Status: "active",
		}

		err := service.RemoveEnvironment("test")

		require.NoError(t, err)
		assert.NotContains(t, service.environments, "test")
	})

	t.Run("remove non-existing environment", func(t *testing.T) {
		err := service.RemoveEnvironment("nonexistent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSyncService_checkURLHealth(t *testing.T) {
	service := NewSyncService(nil)

	t.Run("healthy URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		healthy, err := service.checkURLHealth(server.URL)

		assert.True(t, healthy)
		assert.NoError(t, err)
	})

	t.Run("unhealthy URL - 500 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		healthy, err := service.checkURLHealth(server.URL)

		assert.False(t, healthy)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unhealthy status code")
	})

	t.Run("unreachable URL", func(t *testing.T) {
		healthy, err := service.checkURLHealth("http://invalid.test")

		assert.False(t, healthy)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect")
	})

	t.Run("redirect status codes", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMovedPermanently)
		}))
		defer server.Close()

		healthy, err := service.checkURLHealth(server.URL)

		assert.True(t, healthy)
		assert.NoError(t, err)
	})
}

func TestSyncService_getHealthMessage(t *testing.T) {
	service := NewSyncService(nil)

	tests := []struct {
		name            string
		frontendHealthy bool
		backendHealthy  bool
		expectedMessage string
	}{
		{
			name:            "both healthy",
			frontendHealthy: true,
			backendHealthy:  true,
			expectedMessage: "All services are healthy and connected",
		},
		{
			name:            "both unhealthy",
			frontendHealthy: false,
			backendHealthy:  false,
			expectedMessage: "Both frontend and backend services are unreachable",
		},
		{
			name:            "frontend unhealthy",
			frontendHealthy: false,
			backendHealthy:  true,
			expectedMessage: "Frontend service is unreachable",
		},
		{
			name:            "backend unhealthy",
			frontendHealthy: true,
			backendHealthy:  false,
			expectedMessage: "Backend service is unreachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := service.getHealthMessage(tt.frontendHealthy, tt.backendHealthy)
			assert.Equal(t, tt.expectedMessage, message)
		})
	}
}
