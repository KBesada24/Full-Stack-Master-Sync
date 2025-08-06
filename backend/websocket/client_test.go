package websocket

import (
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	hub := NewHub()
	userID := "test-user"

	// We can't easily mock websocket.Conn, so we'll test with nil for basic structure
	client := &Client{
		ID:       "test-client-id",
		conn:     nil, // Would be a real connection in practice
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   userID,
		LastSeen: time.Now(),
	}

	assert.NotNil(t, client)
	assert.Equal(t, "test-client-id", client.ID)
	assert.Equal(t, userID, client.UserID)
	assert.Equal(t, hub, client.hub)
	assert.NotNil(t, client.send)
	assert.Equal(t, 256, cap(client.send))
}

func TestClient_SendMessage(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:       "test-client-id",
		conn:     nil,
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Test sending a message
	testData := map[string]interface{}{"test": "data"}
	client.SendMessage("test_message", testData)

	// Check if message was added to send channel
	select {
	case msg := <-client.send:
		assert.Equal(t, "test_message", msg.Type)
		assert.Equal(t, testData, msg.Data)
		assert.Equal(t, client.ID, msg.ClientID)
		assert.NotZero(t, msg.Timestamp)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message was not sent to channel")
	}
}

func TestClient_SendMessage_FullChannel(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:       "test-client-id",
		conn:     nil,
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Fill the send channel
	for i := 0; i < 256; i++ {
		client.send <- models.WSMessage{Type: "fill", Data: i}
	}

	// Try to send another message (should not block)
	testData := map[string]interface{}{"test": "data"}
	client.SendMessage("test_message", testData)

	// The message should be dropped, so the channel should still be full
	assert.Equal(t, 256, len(client.send))
}

func TestClient_IsAlive(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:       "test-client-id",
		conn:     nil,
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Client should be alive initially
	assert.True(t, client.IsAlive())

	// Set LastSeen to old time
	client.LastSeen = time.Now().Add(-2 * pongWait)
	assert.False(t, client.IsAlive())
}

func TestClient_HandleMessage_Heartbeat(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:       "test-client-id",
		conn:     nil,
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Create heartbeat message
	heartbeatMsg := models.WSMessage{
		Type:      "heartbeat",
		Data:      map[string]interface{}{"ping": "test"},
		Timestamp: time.Now(),
		ClientID:  client.ID,
	}

	// Handle the message
	client.handleMessage(heartbeatMsg)

	// Check if pong response was sent
	select {
	case response := <-client.send:
		assert.Equal(t, "heartbeat", response.Type)
		assert.Equal(t, client.ID, response.ClientID)
		responseData, ok := response.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "pong", responseData["status"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Heartbeat response was not sent")
	}
}

func TestClient_HandleMessage_Connect(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:       "test-client-id",
		conn:     nil,
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Create connect message
	connectMsg := models.WSMessage{
		Type:      "connect",
		Data:      map[string]interface{}{"status": "connecting"},
		Timestamp: time.Now(),
		ClientID:  client.ID,
	}

	// Handle the message (should just log, no response expected)
	client.handleMessage(connectMsg)

	// No message should be sent to the send channel
	select {
	case <-client.send:
		t.Fatal("No message should be sent for connect acknowledgment")
	case <-time.After(50 * time.Millisecond):
		// This is expected
	}
}

func TestClient_HandleMessage_Disconnect(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		ID:       "test-client-id",
		conn:     nil,
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Register client first
	hub.RegisterClient(client)
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered
	assert.Equal(t, 1, hub.GetConnectedClients())

	// Create disconnect message
	disconnectMsg := models.WSMessage{
		Type:      "disconnect",
		Data:      map[string]interface{}{"reason": "user_requested"},
		Timestamp: time.Now(),
		ClientID:  client.ID,
	}

	// Handle the message
	client.handleMessage(disconnectMsg)
	time.Sleep(10 * time.Millisecond)

	// Client should be unregistered
	assert.Equal(t, 0, hub.GetConnectedClients())
}

func TestClient_HandleMessage_Unknown(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:       "test-client-id",
		conn:     nil,
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Create unknown message type
	unknownMsg := models.WSMessage{
		Type:      "unknown_type",
		Data:      map[string]interface{}{"test": "data"},
		Timestamp: time.Now(),
		ClientID:  client.ID,
	}

	// Handle the message (should just log, no response expected)
	client.handleMessage(unknownMsg)

	// No message should be sent to the send channel
	select {
	case <-client.send:
		t.Fatal("No message should be sent for unknown message type")
	case <-time.After(50 * time.Millisecond):
		// This is expected
	}
}

func TestIsValidMessageType(t *testing.T) {
	validTypes := []string{
		"sync_status_update",
		"test_progress",
		"log_alert",
		"ai_suggestion_ready",
		"connect",
		"disconnect",
		"heartbeat",
	}

	for _, msgType := range validTypes {
		assert.True(t, isValidMessageType(msgType), "Message type %s should be valid", msgType)
	}

	invalidTypes := []string{
		"invalid_type",
		"random_message",
		"",
		"SYNC_STATUS_UPDATE", // case sensitive
	}

	for _, msgType := range invalidTypes {
		assert.False(t, isValidMessageType(msgType), "Message type %s should be invalid", msgType)
	}
}

func TestClient_Constants(t *testing.T) {
	// Test that constants are set to reasonable values
	assert.Equal(t, 10*time.Second, writeWait)
	assert.Equal(t, 60*time.Second, pongWait)
	assert.Equal(t, (pongWait*9)/10, pingPeriod)
	assert.Equal(t, int64(512), int64(maxMessageSize))

	// Ensure pingPeriod is less than pongWait
	assert.True(t, pingPeriod < pongWait)
}
