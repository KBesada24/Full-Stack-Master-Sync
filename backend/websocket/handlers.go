package websocket

import (
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// Global hub instance
var GlobalHub *Hub

// InitializeHub initializes the global WebSocket hub
func InitializeHub() {
	GlobalHub = NewHub()
	go GlobalHub.Run()

	logger := utils.GetLogger()
	logger.Info("WebSocket hub initialized and started", map[string]interface{}{
		"status": "running",
	})
}

// WebSocketUpgrade handles WebSocket upgrade requests
func WebSocketUpgrade(c *fiber.Ctx) error {
	// Check if the request is a WebSocket upgrade
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}

	return utils.ErrorResponse(c, fiber.StatusUpgradeRequired, "WEBSOCKET_REQUIRED", "WebSocket upgrade required", nil)
}

// WebSocketHandler handles WebSocket connections
func WebSocketHandler(c *websocket.Conn) {
	logger := utils.GetLogger()

	// Get user ID from query parameters or headers (for now, use a default)
	userID := c.Query("user_id", "anonymous")

	// Create new client
	client := NewClient(c, GlobalHub, userID)

	// Register client with hub
	GlobalHub.RegisterClient(client)

	logger.Info("New WebSocket connection established", map[string]interface{}{
		"client_id":   client.ID,
		"user_id":     userID,
		"remote_addr": c.RemoteAddr().String(),
	})

	// Start client pumps in separate goroutines
	go client.WritePump()
	client.ReadPump() // This blocks until connection is closed
}

// GetWebSocketStats returns statistics about WebSocket connections
func GetWebSocketStats() map[string]interface{} {
	if GlobalHub == nil {
		return map[string]interface{}{
			"status":            "not_initialized",
			"connected_clients": 0,
		}
	}

	return map[string]interface{}{
		"status":            "running",
		"connected_clients": GlobalHub.GetConnectedClients(),
		"client_ids":        GlobalHub.GetClientIDs(),
	}
}

// BroadcastSyncUpdate broadcasts a sync status update to all clients
func BroadcastSyncUpdate(data interface{}) {
	if GlobalHub != nil {
		GlobalHub.BroadcastToAll("sync_status_update", data)
	}
}

// BroadcastTestProgress broadcasts test progress to all clients
func BroadcastTestProgress(data interface{}) {
	if GlobalHub != nil {
		GlobalHub.BroadcastToAll("test_progress", data)
	}
}

// BroadcastLogAlert broadcasts a log alert to all clients
func BroadcastLogAlert(data interface{}) {
	if GlobalHub != nil {
		GlobalHub.BroadcastToAll("log_alert", data)
	}
}

// BroadcastAISuggestionReady broadcasts AI suggestion ready notification to all clients
func BroadcastAISuggestionReady(data interface{}) {
	if GlobalHub != nil {
		GlobalHub.BroadcastToAll("ai_suggestion_ready", data)
	}
}

// BroadcastToClient sends a message to a specific client
func BroadcastToClient(clientID, msgType string, data interface{}) {
	if GlobalHub != nil {
		GlobalHub.BroadcastToClient(clientID, msgType, data)
	}
}
