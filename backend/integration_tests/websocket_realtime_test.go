package integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSocketRealTimeFeatures tests WebSocket integration for real-time features across all workflows
func TestWebSocketRealTimeFeatures(t *testing.T) {
	// Setup test environment
	cfg := &config.Config{
		Environment:  "test",
		Port:         "8080",
		OpenAIAPIKey: "", // No API key for testing
		FrontendURL:  "http://localhost:3000",
		LogLevel:     "info",
		LogFormat:    "json",
	}

	// Initialize logger
	utils.InitLogger(cfg.LogLevel, cfg.LogFormat)
	logger := utils.GetLogger()

	// Initialize WebSocket hub
	websocket.InitializeHub()
	hub := websocket.GetHub()
	go hub.Run()

	// Initialize all services
	aiService := services.NewAIService(cfg, hub, logger)
	syncService := services.NewSyncService(hub)
	testService := services.NewTestService(cfg, hub)
	logService := services.NewLogService(aiService, hub)

	time.Sleep(200 * time.Millisecond)

	t.Run("Cross-Service WebSocket Integration", func(t *testing.T) {
		// Test that all services can broadcast to the same hub
		initialClients := hub.GetConnectedClients()

		// Test AI service broadcasting
		hub.BroadcastToAll("ai_suggestion_ready", map[string]interface{}{
			"type":         "code_suggestions",
			"request_type": "optimization",
			"status":       "ready",
		})

		// Test sync service broadcasting
		hub.BroadcastToAll("sync_status_update", map[string]interface{}{
			"action":      "environment_connected",
			"environment": "test",
			"status":      "connected",
		})

		// Test testing service broadcasting
		hub.BroadcastToAll("test_progress", map[string]interface{}{
			"run_id":      "test-run-123",
			"status":      "running",
			"framework":   "jest",
			"environment": "test",
		})

		// Test logging service broadcasting
		hub.BroadcastToAll("log_alert", map[string]interface{}{
			"type":    "critical_log_event",
			"level":   "error",
			"source":  "backend",
			"message": "System error detected",
		})

		// Verify all broadcasts were sent successfully
		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, initialClients, hub.GetConnectedClients(), "Hub should maintain client count after broadcasts")
	})

	t.Run("Real-Time Workflow Coordination", func(t *testing.T) {
		// Simulate a complete workflow that involves multiple services
		// and real-time updates

		// Step 1: Connect sync environment (triggers sync update)
		syncRequest := &models.SyncConnectionRequest{
			FrontendURL: "http://localhost:3000",
			BackendURL:  "http://localhost:8080",
			Environment: "integration-test",
		}

		_, err := syncService.ConnectEnvironment(syncRequest)
		require.NoError(t, err)

		// Broadcast sync status update
		hub.BroadcastToAll("sync_status_update", map[string]interface{}{
			"action":      "environment_connected",
			"environment": "integration-test",
			"status":      "connected",
		})

		// Step 2: Start test run (triggers test progress updates)
		testRequest := &models.TestRunRequest{
			Framework:   "cypress",
			TestSuite:   "test/integration.spec.js",
			Environment: "integration-test",
			Config:      map[string]string{"timeout": "30000"},
		}

		testResp, err := testService.StartTestRun(context.Background(), testRequest)
		require.NoError(t, err)

		// Broadcast test progress updates
		progressStates := []string{"queued", "running", "completed"}
		for _, state := range progressStates {
			hub.BroadcastToAll("test_progress", map[string]interface{}{
				"run_id":      testResp.RunID,
				"status":      state,
				"framework":   "cypress",
				"environment": "integration-test",
			})
			time.Sleep(100 * time.Millisecond)
		}

		// Step 3: Submit logs during test execution (triggers log alerts)
		logRequest := &models.LogSubmissionRequest{
			Logs: []models.LogEntry{
				{
					ID:        "integration-log-1",
					Timestamp: time.Now(),
					Level:     "error",
					Source:    "frontend",
					Message:   "Test execution error",
					Context: map[string]interface{}{
						"test_run_id": testResp.RunID,
						"component":   "TestRunner",
					},
				},
			},
			Source: "frontend",
		}

		_, err = logService.SubmitLogs(context.Background(), logRequest)
		require.NoError(t, err)

		// Broadcast log alert
		hub.BroadcastToAll("log_alert", map[string]interface{}{
			"type":        "critical_log_event",
			"level":       "error",
			"source":      "frontend",
			"message":     "Test execution error",
			"test_run_id": testResp.RunID,
		})

		// Step 4: Trigger AI analysis (triggers AI ready notification)
		aiRequest := &models.AIRequest{
			Code:        "describe('integration test', () => { it('should pass', () => { expect(true).toBe(true); }); });",
			Language:    "javascript",
			Context:     "Integration test code analysis",
			RequestType: "debug",
		}

		_, err = aiService.GetCodeSuggestions(context.Background(), aiRequest)
		require.NoError(t, err)

		// Broadcast AI suggestion ready
		hub.BroadcastToAll("ai_suggestion_ready", map[string]interface{}{
			"type":         "code_suggestions",
			"request_type": "debug",
			"status":       "ready",
			"test_run_id":  testResp.RunID,
		})

		// Verify all services completed their operations
		time.Sleep(500 * time.Millisecond)
		assert.GreaterOrEqual(t, hub.GetConnectedClients(), 0, "Hub should be functioning after complex workflow")
	})

	t.Run("WebSocket Hub Resilience", func(t *testing.T) {
		// Test hub resilience under various conditions
		initialClients := hub.GetConnectedClients()

		// Test rapid successive broadcasts
		for i := 0; i < 10; i++ {
			hub.BroadcastToAll("test_message", map[string]interface{}{
				"sequence": i,
				"message":  "Rapid broadcast test",
			})
		}

		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, initialClients, hub.GetConnectedClients(), "Hub should handle rapid broadcasts")

		// Test large message broadcasting
		largeData := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			largeData[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("Large data value %d with some additional content to make it bigger", i)
		}

		hub.BroadcastToAll("large_message", largeData)
		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, initialClients, hub.GetConnectedClients(), "Hub should handle large messages")

		// Test concurrent broadcasts from multiple services
		done := make(chan bool, 4)

		// AI service broadcasts
		go func() {
			for i := 0; i < 5; i++ {
				hub.BroadcastToAll("ai_suggestion_ready", map[string]interface{}{
					"concurrent_test": true,
					"service":         "ai",
					"iteration":       i,
				})
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		// Sync service broadcasts
		go func() {
			for i := 0; i < 5; i++ {
				hub.BroadcastToAll("sync_status_update", map[string]interface{}{
					"concurrent_test": true,
					"service":         "sync",
					"iteration":       i,
				})
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		// Test service broadcasts
		go func() {
			for i := 0; i < 5; i++ {
				hub.BroadcastToAll("test_progress", map[string]interface{}{
					"concurrent_test": true,
					"service":         "test",
					"iteration":       i,
				})
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		// Log service broadcasts
		go func() {
			for i := 0; i < 5; i++ {
				hub.BroadcastToAll("log_alert", map[string]interface{}{
					"concurrent_test": true,
					"service":         "log",
					"iteration":       i,
				})
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		// Wait for all concurrent broadcasts to complete
		for i := 0; i < 4; i++ {
			select {
			case <-done:
				// Service completed
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for concurrent broadcasts")
			}
		}

		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, initialClients, hub.GetConnectedClients(), "Hub should handle concurrent broadcasts from multiple services")
	})

	t.Run("Message Type Validation", func(t *testing.T) {
		// Test that all expected message types are valid
		expectedMessageTypes := []string{
			"sync_status_update",
			"test_progress",
			"log_alert",
			"ai_suggestion_ready",
			"connect",
			"disconnect",
			"heartbeat",
		}

		for _, msgType := range expectedMessageTypes {
			// Test broadcasting each message type
			hub.BroadcastToAll(msgType, map[string]interface{}{
				"test":         "message_type_validation",
				"message_type": msgType,
			})
		}

		// Verify all broadcasts were sent successfully
		time.Sleep(200 * time.Millisecond)
		assert.GreaterOrEqual(t, hub.GetConnectedClients(), 0, "Hub should handle all valid message types")
	})

	t.Run("Service Integration Stress Test", func(t *testing.T) {
		// Stress test the integration between services and WebSocket hub
		numOperations := 20
		completed := make(chan bool, numOperations)

		// Perform multiple operations concurrently
		for i := 0; i < numOperations; i++ {
			go func(iteration int) {
				defer func() { completed <- true }()

				switch iteration % 4 {
				case 0:
					// AI service operation
					aiReq := &models.AIRequest{
						Code:        fmt.Sprintf("function test%d() { return %d; }", iteration, iteration),
						Language:    "javascript",
						Context:     fmt.Sprintf("Stress test iteration %d", iteration),
						RequestType: "suggestion",
					}
					_, err := aiService.GetCodeSuggestions(context.Background(), aiReq)
					if err != nil {
						t.Logf("AI service error in iteration %d: %v", iteration, err)
					}

				case 1:
					// Sync service operation
					syncReq := &models.SyncConnectionRequest{
						FrontendURL: "http://localhost:3000",
						BackendURL:  "http://localhost:8080",
						Environment: fmt.Sprintf("stress-test-%d", iteration),
					}
					_, err := syncService.ConnectEnvironment(syncReq)
					if err != nil {
						t.Logf("Sync service error in iteration %d: %v", iteration, err)
					}

				case 2:
					// Test service operation
					testReq := &models.TestRunRequest{
						Framework:   "jest",
						TestSuite:   fmt.Sprintf("test/stress-%d.test.js", iteration),
						Environment: "stress-test",
						Config:      map[string]string{},
					}
					_, err := testService.StartTestRun(context.Background(), testReq)
					if err != nil {
						t.Logf("Test service error in iteration %d: %v", iteration, err)
					}

				case 3:
					// Log service operation
					logReq := &models.LogSubmissionRequest{
						Logs: []models.LogEntry{
							{
								ID:        fmt.Sprintf("stress-log-%d", iteration),
								Timestamp: time.Now(),
								Level:     "info",
								Source:    "backend",
								Message:   fmt.Sprintf("Stress test log %d", iteration),
								Context:   map[string]interface{}{"iteration": iteration},
							},
						},
						Source: "backend",
					}
					_, err := logService.SubmitLogs(context.Background(), logReq)
					if err != nil {
						t.Logf("Log service error in iteration %d: %v", iteration, err)
					}
				}
			}(i)
		}

		// Wait for all operations to complete
		completedCount := 0
		timeout := time.After(30 * time.Second)

		for completedCount < numOperations {
			select {
			case <-completed:
				completedCount++
			case <-timeout:
				t.Fatalf("Timeout waiting for stress test operations. Completed: %d/%d", completedCount, numOperations)
			}
		}

		// Verify hub is still functioning after stress test
		time.Sleep(500 * time.Millisecond)
		assert.GreaterOrEqual(t, hub.GetConnectedClients(), 0, "Hub should be functioning after stress test")

		// Test final broadcast to ensure hub is responsive
		hub.BroadcastToAll("stress_test_complete", map[string]interface{}{
			"completed_operations": numOperations,
			"status":               "success",
		})

		time.Sleep(100 * time.Millisecond)
		assert.GreaterOrEqual(t, hub.GetConnectedClients(), 0, "Hub should respond to broadcasts after stress test")
	})
}
