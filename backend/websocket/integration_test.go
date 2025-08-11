package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/config"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to consume messages until we find the expected type
func waitForMessageType(t *testing.T, client *Client, expectedType string, timeout time.Duration) models.WSMessage {
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-client.send:
			if msg.Type == expectedType {
				return msg
			}
			// Continue waiting for the expected message type
		case <-deadline:
			t.Fatalf("Timeout waiting for message type: %s", expectedType)
		}
	}
}

// TestWebSocketServiceIntegration tests the WebSocket integration with services
func TestWebSocketServiceIntegration(t *testing.T) {
	// Initialize test configuration
	cfg := &config.Config{
		Environment:  "test",
		Port:         "8080",
		OpenAIAPIKey: "", // No API key for testing
		FrontendURL:  "http://localhost:3000",
	}

	// Initialize WebSocket hub
	hub := NewHub()
	go hub.Run()

	// Initialize services with WebSocket integration
	aiService := services.NewAIService(cfg, hub, nil) // nil logger for testing
	syncService := services.NewSyncService(hub)
	testService := services.NewTestService(cfg, hub)
	logService := services.NewLogService(aiService, hub)

	// Create mock client to receive messages
	mockClient := &Client{
		ID:     "test-client",
		send:   make(chan models.WSMessage, 20), // Larger buffer
		hub:    hub,
		UserID: "test-user",
	}

	// Register mock client
	hub.RegisterClient(mockClient)
	time.Sleep(100 * time.Millisecond) // Wait for registration

	// Test sync status broadcasting
	t.Run("Sync Status Broadcasting", func(t *testing.T) {
		// Trigger sync connection
		req := &models.SyncConnectionRequest{
			FrontendURL: "http://localhost:3000",
			BackendURL:  "http://localhost:8080",
			Environment: "test",
		}

		_, err := syncService.ConnectEnvironment(req)
		require.NoError(t, err)

		// Wait for sync status update
		msg := waitForMessageType(t, mockClient, "sync_status_update", 5*time.Second)

		// Verify message data
		data, ok := msg.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "sync_status_change", data["type"])
		assert.Contains(t, []string{"connected", "error"}, data["status"])
	})

	// Test AI suggestion broadcasting
	t.Run("AI Suggestion Broadcasting", func(t *testing.T) {
		// Trigger AI suggestion (will use fallback since no API key)
		aiReq := &models.AIRequest{
			Code:        "console.log('hello world');",
			Language:    "javascript",
			Context:     "Test code",
			RequestType: "suggestion",
		}

		_, err := aiService.GetCodeSuggestions(context.Background(), aiReq)
		require.NoError(t, err)

		// Wait for AI suggestion ready notification
		msg := waitForMessageType(t, mockClient, "ai_suggestion_ready", 5*time.Second)

		// Verify message data
		data, ok := msg.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "code_suggestions", data["type"])
		assert.Equal(t, "suggestion", data["request_type"])
		assert.Equal(t, "ready", data["status"])
	})

	// Test log alert broadcasting
	t.Run("Log Alert Broadcasting", func(t *testing.T) {
		// Submit critical log entry
		logReq := &models.LogSubmissionRequest{
			Logs: []models.LogEntry{
				{
					ID:        "test-log-1",
					Timestamp: time.Now(),
					Level:     "error",
					Source:    "backend",
					Message:   "Critical system error occurred",
					Context:   map[string]interface{}{"component": "test"},
				},
			},
			Source: "backend",
		}

		_, err := logService.SubmitLogs(context.Background(), logReq)
		require.NoError(t, err)

		// Wait for log alert
		msg := waitForMessageType(t, mockClient, "log_alert", 5*time.Second)

		// Verify message data
		data, ok := msg.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "critical_log_event", data["type"])
		assert.Equal(t, "error", data["level"])
		assert.Equal(t, "backend", data["source"])
	})

	// Test test progress broadcasting
	t.Run("Test Progress Broadcasting", func(t *testing.T) {
		// Start a test run
		testReq := &models.TestRunRequest{
			Framework:   "jest",
			TestSuite:   "test/sample.test.js",
			Environment: "test",
			Config:      map[string]string{}, // Remove workDir to avoid path issues
		}

		resp, err := testService.StartTestRun(context.Background(), testReq)
		require.NoError(t, err)

		// Wait for test progress updates
		progressUpdates := 0
		timeout := time.After(10 * time.Second)

		for progressUpdates < 2 { // Expect at least queued and running/completed
			select {
			case msg := <-mockClient.send:
				if msg.Type == "test_progress" {
					progressUpdates++

					// Verify message data
					data, ok := msg.Data.(map[string]interface{})
					require.True(t, ok)
					assert.Equal(t, resp.RunID, data["run_id"])
					assert.Contains(t, []string{"queued", "running", "completed", "failed"}, data["status"])
					assert.Equal(t, "jest", data["framework"])
					assert.Equal(t, "test", data["environment"])
				}
			case <-timeout:
				t.Fatalf("Timeout waiting for test progress updates. Received %d updates", progressUpdates)
			}
		}

		assert.GreaterOrEqual(t, progressUpdates, 2, "Should receive at least 2 progress updates")
	})
}

// TestWebSocketMessageValidation tests WebSocket message validation
func TestWebSocketMessageValidation(t *testing.T) {
	// Test valid message types
	validMessages := []models.WSMessage{
		{Type: "sync_status_update", Data: map[string]interface{}{"test": "data"}, ClientID: "test", Timestamp: time.Now()},
		{Type: "test_progress", Data: map[string]interface{}{"test": "data"}, ClientID: "test", Timestamp: time.Now()},
		{Type: "log_alert", Data: map[string]interface{}{"test": "data"}, ClientID: "test", Timestamp: time.Now()},
		{Type: "ai_suggestion_ready", Data: map[string]interface{}{"test": "data"}, ClientID: "test", Timestamp: time.Now()},
		{Type: "connect", Data: map[string]interface{}{"test": "data"}, ClientID: "test", Timestamp: time.Now()},
		{Type: "disconnect", Data: map[string]interface{}{"test": "data"}, ClientID: "test", Timestamp: time.Now()},
		{Type: "heartbeat", Data: map[string]interface{}{"test": "data"}, ClientID: "test", Timestamp: time.Now()},
	}

	for _, msg := range validMessages {
		msgBytes, err := json.Marshal(msg)
		require.NoError(t, err)

		// Process the message
		var receivedMsg models.WSMessage
		err = json.Unmarshal(msgBytes, &receivedMsg)
		require.NoError(t, err)

		// Validate message type
		assert.True(t, isValidMessageType(receivedMsg.Type), "Message type %s should be valid", receivedMsg.Type)
	}

	// Test invalid message types
	invalidMessages := []string{"invalid_type", "unknown_message", ""}

	for _, msgType := range invalidMessages {
		assert.False(t, isValidMessageType(msgType), "Message type %s should be invalid", msgType)
	}
}

// TestWebSocketHubOperations tests WebSocket hub operations
func TestWebSocketHubOperations(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create mock clients
	client1 := &Client{
		ID:     "client-1",
		send:   make(chan models.WSMessage, 20), // Larger buffer
		hub:    hub,
		UserID: "user-1",
	}

	client2 := &Client{
		ID:     "client-2",
		send:   make(chan models.WSMessage, 20), // Larger buffer
		hub:    hub,
		UserID: "user-2",
	}

	// Register clients
	hub.RegisterClient(client1)
	hub.RegisterClient(client2)

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	// Test client count
	assert.Equal(t, 2, hub.GetConnectedClients())

	// Test client IDs
	clientIDs := hub.GetClientIDs()
	assert.Len(t, clientIDs, 2)
	assert.Contains(t, clientIDs, "client-1")
	assert.Contains(t, clientIDs, "client-2")

	// Test broadcasting to all clients
	hub.BroadcastToAll("test_message", map[string]interface{}{"test": "broadcast"})

	// Wait for message delivery
	time.Sleep(100 * time.Millisecond)

	// Verify both clients received the message (ignoring connect messages)
	msg1 := waitForMessageType(t, client1, "test_message", 1*time.Second)
	assert.Equal(t, "test_message", msg1.Type)

	msg2 := waitForMessageType(t, client2, "test_message", 1*time.Second)
	assert.Equal(t, "test_message", msg2.Type)

	// Test broadcasting to specific client
	hub.BroadcastToClient("client-1", "specific_message", map[string]interface{}{"target": "client-1"})

	// Wait for message delivery
	time.Sleep(100 * time.Millisecond)

	// Verify only client 1 received the message
	msg := waitForMessageType(t, client1, "specific_message", 1*time.Second)
	assert.Equal(t, "specific_message", msg.Type)

	// Verify client 2 did not receive the specific message
	select {
	case msg := <-client2.send:
		if msg.Type == "specific_message" {
			t.Fatal("Client 2 should not have received targeted message")
		}
		// If it's another message type, that's fine
	case <-time.After(100 * time.Millisecond):
		// Expected - no specific message received
	}

	// Test client unregistration
	hub.UnregisterClient(client1)
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, hub.GetConnectedClients())
	clientIDs = hub.GetClientIDs()
	assert.Len(t, clientIDs, 1)
	assert.Contains(t, clientIDs, "client-2")
}

// TestRealTimeNotifications tests real-time notification scenarios
func TestRealTimeNotifications(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create test client
	client := &Client{
		ID:     "notification-test-client",
		send:   make(chan models.WSMessage, 30), // Large buffer
		hub:    hub,
		UserID: "test-user",
	}

	hub.RegisterClient(client)
	time.Sleep(100 * time.Millisecond)

	// Test multiple notification types
	notifications := []struct {
		msgType string
		data    map[string]interface{}
	}{
		{
			msgType: "sync_status_update",
			data: map[string]interface{}{
				"type":      "sync_status_change",
				"status":    "connected",
				"connected": true,
			},
		},
		{
			msgType: "test_progress",
			data: map[string]interface{}{
				"run_id":    "test-run-123",
				"status":    "running",
				"framework": "cypress",
			},
		},
		{
			msgType: "log_alert",
			data: map[string]interface{}{
				"type":    "critical_log_event",
				"level":   "error",
				"message": "System failure detected",
			},
		},
		{
			msgType: "ai_suggestion_ready",
			data: map[string]interface{}{
				"type":         "code_suggestions",
				"request_type": "optimization",
				"status":       "ready",
			},
		},
	}

	// Send all notifications
	for _, notification := range notifications {
		hub.BroadcastToAll(notification.msgType, notification.data)
	}

	// Verify all notifications were received
	receivedTypes := make(map[string]bool)
	timeout := time.After(5 * time.Second)

	for len(receivedTypes) < len(notifications) {
		select {
		case msg := <-client.send:
			// Check if this is one of our expected notifications
			for _, notification := range notifications {
				if msg.Type == notification.msgType {
					receivedTypes[msg.Type] = true
					break
				}
			}

		case <-timeout:
			t.Fatalf("Timeout waiting for notifications. Received %d out of %d", len(receivedTypes), len(notifications))
		}
	}

	assert.Equal(t, len(notifications), len(receivedTypes), "Should receive all notification types")

	// Verify all expected types were received
	for _, notification := range notifications {
		assert.True(t, receivedTypes[notification.msgType], "Should have received %s notification", notification.msgType)
	}
}
