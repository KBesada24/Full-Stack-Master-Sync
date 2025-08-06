package websocket

import (
	"log"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
)

// Hub manages WebSocket connections and message broadcasting
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan models.WSMessage
	register   chan *Client
	unregister chan *Client
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan models.WSMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the WebSocket hub and handles client connections and messages
func (h *Hub) Run() {
	logger := utils.GetLogger()

	for {
		select {
		case client := <-h.register:
			// Register new client
			h.clients[client] = true
			logger.Info("WebSocket client connected", map[string]interface{}{
				"client_id":     client.ID,
				"user_id":       client.UserID,
				"total_clients": len(h.clients),
			})

			// Send welcome message to the new client
			welcomeMsg := models.WSMessage{
				Type:      "connect",
				Data:      map[string]interface{}{"status": "connected", "client_id": client.ID},
				Timestamp: time.Now(),
				ClientID:  client.ID,
			}

			select {
			case client.send <- welcomeMsg:
			default:
				close(client.send)
				delete(h.clients, client)
			}

		case client := <-h.unregister:
			// Unregister client
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				logger.Info("WebSocket client disconnected", map[string]interface{}{
					"client_id":     client.ID,
					"user_id":       client.UserID,
					"total_clients": len(h.clients),
				})
			}

		case message := <-h.broadcast:
			// Broadcast message to all clients
			logger.Debug("Broadcasting WebSocket message", map[string]interface{}{
				"type":       message.Type,
				"client_id":  message.ClientID,
				"recipients": len(h.clients),
			})

			// Send message to all connected clients
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is blocked, remove the client
					close(client.send)
					delete(h.clients, client)
					logger.Warn("Removed unresponsive WebSocket client", map[string]interface{}{
						"client_id": client.ID,
					})
				}
			}
		}
	}
}

// BroadcastToAll sends a message to all connected clients
func (h *Hub) BroadcastToAll(msgType string, data interface{}) {
	message := models.WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		ClientID:  "server",
	}

	select {
	case h.broadcast <- message:
	default:
		log.Printf("Warning: Broadcast channel is full, message dropped")
	}
}

// BroadcastToClient sends a message to a specific client
func (h *Hub) BroadcastToClient(clientID string, msgType string, data interface{}) {
	message := models.WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		ClientID:  clientID,
	}

	// Find the specific client and send the message
	for client := range h.clients {
		if client.ID == clientID {
			select {
			case client.send <- message:
				return
			default:
				// Client's send channel is blocked, remove the client
				close(client.send)
				delete(h.clients, client)
				utils.GetLogger().Warn("Removed unresponsive WebSocket client during targeted broadcast", map[string]interface{}{
					"client_id": clientID,
				})
			}
			return
		}
	}
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	return len(h.clients)
}

// GetClientIDs returns a list of all connected client IDs
func (h *Hub) GetClientIDs() []string {
	clientIDs := make([]string, 0, len(h.clients))
	for client := range h.clients {
		clientIDs = append(clientIDs, client.ID)
	}
	return clientIDs
}

// RegisterClient registers a new client with the hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client from the hub
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}
