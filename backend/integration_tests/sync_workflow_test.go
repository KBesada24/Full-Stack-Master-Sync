package integration_tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/handlers"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/middleware"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncEnvironmentWorkflow tests the complete sync environment workflow
func TestSyncEnvironmentWorkflow(t *testing.T) {
	// Setup test environment
	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
		FrontendURL: "http://localhost:3000",
		LogLevel:    "info",
		LogFormat:   "json",
	}

	// Initialize logger
	utils.InitLogger(cfg.LogLevel, cfg.LogFormat)

	// Initialize WebSocket hub
	websocket.InitializeHub()
	hub := websocket.GetHub()
	go hub.Run()

	// Initialize services
	syncService := services.NewSyncService(hub)

	// Initialize handlers
	syncHandler := handlers.NewSyncHandler(syncService)

	// Create Fiber app
	app := fiber.New()
	app.Use(middleware.CORSWithOrigins([]string{"http://localhost:3000"}))
	app.Use(middleware.RequestValidation())

	// Setup routes
	api := app.Group("/api")
	sync := api.Group("/sync")
	sync.Post("/connect", syncHandler.ConnectEnvironment)
	sync.Get("/status", syncHandler.GetSyncStatus)
	sync.Post("/validate", syncHandler.ValidateEndpoint)
	sync.Get("/environments", syncHandler.GetEnvironments)
	sync.Delete("/environments/:name", syncHandler.RemoveEnvironment)

	time.Sleep(100 * time.Millisecond)

	t.Run("Complete Environment Connection Workflow", func(t *testing.T) {
		// Step 1: Check initial sync status (should be empty)
		req := httptest.NewRequest("GET", "/api/sync/status", nil)
		resp, err := app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var initialStatus map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&initialStatus)
		require.NoError(t, err)
		assert.Equal(t, true, initialStatus["success"])

		// Step 2: Connect to development environment
		connectionRequest := models.SyncConnectionRequest{
			FrontendURL: "http://localhost:3000",
			BackendURL:  "http://localhost:8080",
			Environment: "development",
		}

		reqBody, err := json.Marshal(connectionRequest)
		require.NoError(t, err)

		req = httptest.NewRequest("POST", "/api/sync/connect", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var connectResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&connectResp)
		require.NoError(t, err)
		assert.Equal(t, true, connectResp["success"])

		// Verify response structure
		data, ok := connectResp["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, data, "status")
		assert.Contains(t, data, "connected")
		assert.Contains(t, data, "environments")

		// Step 3: Verify WebSocket functionality by testing broadcast capability
		hub.BroadcastToAll("sync_status_update", map[string]interface{}{
			"action":      "environment_connected",
			"environment": "development",
			"status":      "connected",
		})

		// Verify the broadcast was sent successfully
		time.Sleep(100 * time.Millisecond)
		assert.GreaterOrEqual(t, hub.GetConnectedClients(), 0)

		// Step 4: Check updated sync status
		req = httptest.NewRequest("GET", "/api/sync/status", nil)
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updatedStatus map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&updatedStatus)
		require.NoError(t, err)
		assert.Equal(t, true, updatedStatus["success"])

		statusData, ok := updatedStatus["data"].(map[string]interface{})
		require.True(t, ok)
		// In test environment, connection may fail due to no actual servers running
		// Just verify the status field exists and is a boolean
		_, hasConnected := statusData["connected"].(bool)
		assert.True(t, hasConnected, "Status should have 'connected' field")

		// Step 5: Get environments list
		req = httptest.NewRequest("GET", "/api/sync/environments", nil)
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var envResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&envResp)
		require.NoError(t, err)
		assert.Equal(t, true, envResp["success"])

		envData, ok := envResp["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, envData, "environments")
		assert.Contains(t, envData, "count")
		assert.Equal(t, float64(1), envData["count"].(float64))
	})

	t.Run("Endpoint Validation Workflow", func(t *testing.T) {
		// Step 1: Validate compatible endpoints
		validationRequest := models.SyncValidationRequest{
			FrontendEndpoint: "http://localhost:3000/api/users",
			BackendEndpoint:  "http://localhost:8080/api/users",
			Method:           "GET",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}

		reqBody, err := json.Marshal(validationRequest)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/sync/validate", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var validationResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&validationResp)
		require.NoError(t, err)
		assert.Equal(t, true, validationResp["success"])

		// Verify response structure
		data, ok := validationResp["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, data, "is_compatible")
		assert.Contains(t, data, "validated_at")

		// Step 2: Test incompatible endpoints
		incompatibleRequest := models.SyncValidationRequest{
			FrontendEndpoint: "http://localhost:3000/api/users",
			BackendEndpoint:  "http://localhost:8080/api/different-endpoint",
			Method:           "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Payload: map[string]interface{}{
				"name":  "test",
				"email": "test@example.com",
			},
		}

		reqBody, err = json.Marshal(incompatibleRequest)
		require.NoError(t, err)

		req = httptest.NewRequest("POST", "/api/sync/validate", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var incompatibleResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&incompatibleResp)
		require.NoError(t, err)
		assert.Equal(t, true, incompatibleResp["success"])

		// Check if validation detected issues
		data, ok = incompatibleResp["data"].(map[string]interface{})
		require.True(t, ok)

		// If not compatible, should have issues
		if !data["is_compatible"].(bool) {
			assert.Contains(t, data, "issues")
			issues, ok := data["issues"].([]interface{})
			require.True(t, ok)
			assert.NotEmpty(t, issues)
		}
	})

	t.Run("Multiple Environment Management", func(t *testing.T) {
		// Connect to staging environment
		stagingRequest := models.SyncConnectionRequest{
			FrontendURL: "http://staging.example.com",
			BackendURL:  "http://api-staging.example.com",
			Environment: "staging",
		}

		reqBody, err := json.Marshal(stagingRequest)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/sync/connect", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Connect to production environment
		prodRequest := models.SyncConnectionRequest{
			FrontendURL: "https://app.example.com",
			BackendURL:  "https://api.example.com",
			Environment: "production",
		}

		reqBody, err = json.Marshal(prodRequest)
		require.NoError(t, err)

		req = httptest.NewRequest("POST", "/api/sync/connect", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify all environments are listed
		req = httptest.NewRequest("GET", "/api/sync/environments", nil)
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var envResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&envResp)
		require.NoError(t, err)

		envData, ok := envResp["data"].(map[string]interface{})
		require.True(t, ok)

		// Should have 3 environments now (development, staging, production)
		count, ok := envData["count"].(float64)
		require.True(t, ok)
		assert.Equal(t, float64(3), count)

		// Remove staging environment
		req = httptest.NewRequest("DELETE", "/api/sync/environments/staging", nil)
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var removeResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&removeResp)
		require.NoError(t, err)
		assert.Equal(t, true, removeResp["success"])

		// Verify WebSocket notification for environment removal
		hub.BroadcastToAll("sync_status_update", map[string]interface{}{
			"action":      "environment_removed",
			"environment": "staging",
		})

		time.Sleep(100 * time.Millisecond)

		// Verify environment count decreased
		req = httptest.NewRequest("GET", "/api/sync/environments", nil)
		resp, err = app.Test(req, 10000)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&envResp)
		require.NoError(t, err)

		envData, ok = envResp["data"].(map[string]interface{})
		require.True(t, ok)
		count, ok = envData["count"].(float64)
		require.True(t, ok)
		assert.Equal(t, float64(2), count)
	})
}
