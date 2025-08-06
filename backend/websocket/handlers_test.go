package websocket

import (
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/stretchr/testify/assert"
)

func TestInitializeHub(t *testing.T) {
	// Reset global hub
	GlobalHub = nil

	// Initialize hub
	InitializeHub()

	// Verify hub was created and started
	assert.NotNil(t, GlobalHub)
	assert.NotNil(t, GlobalHub.clients)
	assert.NotNil(t, GlobalHub.broadcast)
	assert.NotNil(t, GlobalHub.register)
	assert.NotNil(t, GlobalHub.unregister)
}

func TestGetWebSocketStats_NotInitialized(t *testing.T) {
	// Reset global hub
	GlobalHub = nil

	stats := GetWebSocketStats()

	assert.Equal(t, "not_initialized", stats["status"])
	assert.Equal(t, 0, stats["connected_clients"])
}

func TestGetWebSocketStats_Running(t *testing.T) {
	// Initialize hub
	InitializeHub()

	stats := GetWebSocketStats()

	assert.Equal(t, "running", stats["status"])
	assert.Equal(t, 0, stats["connected_clients"])
	assert.NotNil(t, stats["client_ids"])
}

func TestBroadcastSyncUpdate(t *testing.T) {
	// Initialize hub
	InitializeHub()
	go GlobalHub.Run()

	testData := map[string]interface{}{
		"status": "connected",
		"url":    "http://localhost:3000",
	}

	// This should not panic
	BroadcastSyncUpdate(testData)

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)

	// Since no clients are connected, the message should be processed but not delivered
	// We just verify the function doesn't panic
}

func TestBroadcastTestProgress(t *testing.T) {
	// Initialize hub
	InitializeHub()
	go GlobalHub.Run()

	testData := map[string]interface{}{
		"test_id":  "test-123",
		"progress": 50,
		"status":   "running",
	}

	// This should not panic
	BroadcastTestProgress(testData)

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)

	// Since no clients are connected, the message should be processed but not delivered
	// We just verify the function doesn't panic
}

func TestBroadcastLogAlert(t *testing.T) {
	// Initialize hub
	InitializeHub()
	go GlobalHub.Run()

	testData := map[string]interface{}{
		"level":   "error",
		"message": "Critical error occurred",
		"source":  "backend",
	}

	// This should not panic
	BroadcastLogAlert(testData)

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)

	// Since no clients are connected, the message should be processed but not delivered
	// We just verify the function doesn't panic
}

func TestBroadcastAISuggestionReady(t *testing.T) {
	// Initialize hub
	InitializeHub()
	go GlobalHub.Run()

	testData := map[string]interface{}{
		"request_id":  "ai-req-123",
		"suggestions": []string{"suggestion1", "suggestion2"},
		"confidence":  0.85,
	}

	// This should not panic
	BroadcastAISuggestionReady(testData)

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)

	// Since no clients are connected, the message should be processed but not delivered
	// We just verify the function doesn't panic
}

func TestBroadcastToClient(t *testing.T) {
	// Initialize hub
	InitializeHub()
	go GlobalHub.Run()

	// Create a test client
	client := &Client{
		ID:       "test-client-1",
		send:     make(chan models.WSMessage, 256),
		hub:      GlobalHub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Register the client
	GlobalHub.RegisterClient(client)
	time.Sleep(10 * time.Millisecond)

	// Clear welcome message
	<-client.send

	testData := map[string]interface{}{
		"message": "Hello specific client",
	}

	// Broadcast to specific client
	BroadcastToClient("test-client-1", "custom_message", testData)

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)

	// Verify client received the message
	select {
	case msg := <-client.send:
		assert.Equal(t, "custom_message", msg.Type)
		assert.Equal(t, testData, msg.Data)
		assert.Equal(t, "test-client-1", msg.ClientID)
	default:
		t.Fatal("Client did not receive the targeted message")
	}
}

func TestBroadcastToClient_NonExistentClient(t *testing.T) {
	// Initialize hub
	InitializeHub()

	testData := map[string]interface{}{
		"message": "Hello non-existent client",
	}

	// This should not panic even for non-existent client
	BroadcastToClient("non-existent-client", "custom_message", testData)
}

func TestBroadcastFunctions_NilHub(t *testing.T) {
	// Reset global hub to nil
	GlobalHub = nil

	// These should not panic even when hub is nil
	BroadcastSyncUpdate(map[string]interface{}{"test": "data"})
	BroadcastTestProgress(map[string]interface{}{"test": "data"})
	BroadcastLogAlert(map[string]interface{}{"test": "data"})
	BroadcastAISuggestionReady(map[string]interface{}{"test": "data"})
	BroadcastToClient("client-id", "message", map[string]interface{}{"test": "data"})

	// No assertions needed - just testing that no panic occurs
}
