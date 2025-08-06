package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/KBesada24/Full-Stack-Master-Sync.git/models"
	"github.com/KBesada24/Full-Stack-Master-Sync.git/utils"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	conn     *websocket.Conn
	send     chan models.WSMessage
	hub      *Hub
	UserID   string
	LastSeen time.Time
}

// NewClient creates a new WebSocket client
func NewClient(conn *websocket.Conn, hub *Hub, userID string) *Client {
	clientID := uuid.New().String()

	return &Client{
		ID:       clientID,
		conn:     conn,
		send:     make(chan models.WSMessage, 256),
		hub:      hub,
		UserID:   userID,
		LastSeen: time.Now(),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	logger := utils.GetLogger()

	defer func() {
		c.hub.UnregisterClient(c)
		c.conn.Close()
	}()

	// Set read deadline and message size limit
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.LastSeen = time.Now()
		return nil
	})

	for {
		// Read message from WebSocket
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error", err, map[string]interface{}{
					"client_id": c.ID,
					"user_id":   c.UserID,
				})
			}
			break
		}

		// Parse the message
		var message models.WSMessage
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			logger.Error("Failed to parse WebSocket message", err, map[string]interface{}{
				"client_id": c.ID,
				"message":   string(messageBytes),
			})
			continue
		}

		// Set client ID and timestamp
		message.ClientID = c.ID
		message.Timestamp = time.Now()
		c.LastSeen = time.Now()

		// Validate message type
		if !isValidMessageType(message.Type) {
			logger.Warn("Invalid WebSocket message type", map[string]interface{}{
				"client_id":    c.ID,
				"message_type": message.Type,
			})
			continue
		}

		// Handle the message
		c.handleMessage(message)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	logger := utils.GetLogger()
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send the message
			messageBytes, err := json.Marshal(message)
			if err != nil {
				logger.Error("Failed to marshal WebSocket message", err, map[string]interface{}{
					"client_id":    c.ID,
					"message_type": message.Type,
				})
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
				logger.Error("Failed to write WebSocket message", err, map[string]interface{}{
					"client_id": c.ID,
				})
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(message models.WSMessage) {
	logger := utils.GetLogger()

	switch message.Type {
	case "heartbeat":
		// Respond to heartbeat with pong
		response := models.WSMessage{
			Type:      "heartbeat",
			Data:      map[string]interface{}{"status": "pong"},
			Timestamp: time.Now(),
			ClientID:  c.ID,
		}

		select {
		case c.send <- response:
		default:
			logger.Warn("Failed to send heartbeat response", map[string]interface{}{
				"client_id": c.ID,
			})
		}

	case "connect":
		// Handle connection acknowledgment
		logger.Info("WebSocket client connection acknowledged", map[string]interface{}{
			"client_id": c.ID,
			"user_id":   c.UserID,
		})

	case "disconnect":
		// Handle graceful disconnect
		logger.Info("WebSocket client requested disconnect", map[string]interface{}{
			"client_id": c.ID,
		})
		c.hub.UnregisterClient(c)

	default:
		// For other message types, just log them for now
		// In future tasks, these will be handled by specific services
		logger.Debug("Received WebSocket message", map[string]interface{}{
			"client_id":    c.ID,
			"message_type": message.Type,
			"data":         message.Data,
		})
	}
}

// isValidMessageType checks if the message type is valid
func isValidMessageType(msgType string) bool {
	validTypes := map[string]bool{
		"sync_status_update":  true,
		"test_progress":       true,
		"log_alert":           true,
		"ai_suggestion_ready": true,
		"connect":             true,
		"disconnect":          true,
		"heartbeat":           true,
	}

	return validTypes[msgType]
}

// SendMessage sends a message to this specific client
func (c *Client) SendMessage(msgType string, data interface{}) {
	message := models.WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		ClientID:  c.ID,
	}

	select {
	case c.send <- message:
	default:
		log.Printf("Warning: Client %s send channel is full, message dropped", c.ID)
	}
}

// IsAlive checks if the client connection is still alive
func (c *Client) IsAlive() bool {
	return time.Since(c.LastSeen) < pongWait
}
