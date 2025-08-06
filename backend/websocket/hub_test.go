package websocket

import (
	"testing"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/stretchr/testify/assert"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.Equal(t, 0, len(hub.clients))
}

func TestHub_RegisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a mock client
	client := &Client{
		ID:       "test-client-1",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Register the client
	hub.RegisterClient(client)

	// Give some time for the hub to process
	time.Sleep(10 * time.Millisecond)

	// Check if client is registered
	assert.Equal(t, 1, len(hub.clients))
	assert.True(t, hub.clients[client])
}

func TestHub_UnregisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a mock client
	client := &Client{
		ID:       "test-client-1",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Register the client first
	hub.RegisterClient(client)
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered
	assert.Equal(t, 1, len(hub.clients))

	// Unregister the client
	hub.UnregisterClient(client)
	time.Sleep(10 * time.Millisecond)

	// Check if client is unregistered
	assert.Equal(t, 0, len(hub.clients))
	assert.False(t, hub.clients[client])
}

func TestHub_BroadcastToAll(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create mock clients
	client1 := &Client{
		ID:       "test-client-1",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user-1",
		LastSeen: time.Now(),
	}

	client2 := &Client{
		ID:       "test-client-2",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user-2",
		LastSeen: time.Now(),
	}

	// Register clients
	hub.RegisterClient(client1)
	hub.RegisterClient(client2)
	time.Sleep(10 * time.Millisecond)

	// Clear welcome messages
	<-client1.send
	<-client2.send

	// Broadcast a message
	testData := map[string]interface{}{"test": "data"}
	hub.BroadcastToAll("test_message", testData)

	// Give some time for the broadcast to process
	time.Sleep(10 * time.Millisecond)

	// Check if both clients received the message
	select {
	case msg1 := <-client1.send:
		assert.Equal(t, "test_message", msg1.Type)
		assert.Equal(t, testData, msg1.Data)
		assert.Equal(t, "server", msg1.ClientID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client 1 did not receive message")
	}

	select {
	case msg2 := <-client2.send:
		assert.Equal(t, "test_message", msg2.Type)
		assert.Equal(t, testData, msg2.Data)
		assert.Equal(t, "server", msg2.ClientID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client 2 did not receive message")
	}
}

func TestHub_BroadcastToClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create mock clients
	client1 := &Client{
		ID:       "test-client-1",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user-1",
		LastSeen: time.Now(),
	}

	client2 := &Client{
		ID:       "test-client-2",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user-2",
		LastSeen: time.Now(),
	}

	// Register clients
	hub.RegisterClient(client1)
	hub.RegisterClient(client2)
	time.Sleep(10 * time.Millisecond)

	// Clear welcome messages
	<-client1.send
	<-client2.send

	// Broadcast to specific client
	testData := map[string]interface{}{"test": "data"}
	hub.BroadcastToClient("test-client-1", "targeted_message", testData)

	// Give some time for the broadcast to process
	time.Sleep(10 * time.Millisecond)

	// Check if only client1 received the message
	select {
	case msg1 := <-client1.send:
		assert.Equal(t, "targeted_message", msg1.Type)
		assert.Equal(t, testData, msg1.Data)
		assert.Equal(t, "test-client-1", msg1.ClientID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client 1 did not receive targeted message")
	}

	// Check that client2 did not receive the message
	select {
	case <-client2.send:
		t.Fatal("Client 2 should not have received the targeted message")
	case <-time.After(50 * time.Millisecond):
		// This is expected - client2 should not receive the message
	}
}

func TestHub_GetConnectedClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Initially no clients
	assert.Equal(t, 0, hub.GetConnectedClients())

	// Add clients
	client1 := &Client{
		ID:       "test-client-1",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user-1",
		LastSeen: time.Now(),
	}

	client2 := &Client{
		ID:       "test-client-2",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user-2",
		LastSeen: time.Now(),
	}

	hub.RegisterClient(client1)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.GetConnectedClients())

	hub.RegisterClient(client2)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 2, hub.GetConnectedClients())

	// Remove a client
	hub.UnregisterClient(client1)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.GetConnectedClients())
}

func TestHub_GetClientIDs(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Initially no clients
	clientIDs := hub.GetClientIDs()
	assert.Equal(t, 0, len(clientIDs))

	// Add clients
	client1 := &Client{
		ID:       "test-client-1",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user-1",
		LastSeen: time.Now(),
	}

	client2 := &Client{
		ID:       "test-client-2",
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   "test-user-2",
		LastSeen: time.Now(),
	}

	hub.RegisterClient(client1)
	hub.RegisterClient(client2)
	time.Sleep(10 * time.Millisecond)

	clientIDs = hub.GetClientIDs()
	assert.Equal(t, 2, len(clientIDs))
	assert.Contains(t, clientIDs, "test-client-1")
	assert.Contains(t, clientIDs, "test-client-2")
}

func TestHub_HandleUnresponsiveClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a client with a small send buffer to simulate blocking
	client := &Client{
		ID:       "test-client-1",
		send:     make(chan models.WSMessage, 1), // Small buffer
		hub:      hub,
		UserID:   "test-user",
		LastSeen: time.Now(),
	}

	// Register the client
	hub.RegisterClient(client)
	time.Sleep(10 * time.Millisecond)

	// Clear the welcome message
	<-client.send

	// Fill the client's send buffer to make it unresponsive
	client.send <- models.WSMessage{Type: "test", Data: "data1"}

	// Try to broadcast a message - this should trigger the unresponsive client removal
	hub.BroadcastToAll("test_message", map[string]interface{}{"test": "data"})

	// Give some time for processing
	time.Sleep(50 * time.Millisecond)

	// The unresponsive client should have been removed
	assert.Equal(t, 0, hub.GetConnectedClients())
}
