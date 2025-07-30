package models

import (
	"time"

	"github.com/gofiber/websocket/v2"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	TraceID string            `json:"trace_id"`
}

// HealthStatus represents the health status of various system components
type HealthStatus struct {
	Frontend bool   `json:"frontend"`
	Backend  bool   `json:"backend"`
	Database bool   `json:"database"`
	Message  string `json:"message"`
}

// WSMessage represents a WebSocket message structure
type WSMessage struct {
	Type      string      `json:"type" validate:"required,oneof=sync_status_update test_progress log_alert ai_suggestion_ready connect disconnect heartbeat"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	ClientID  string      `json:"client_id" validate:"required"`
}

// WSClient represents a WebSocket client connection
type WSClient struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan WSMessage
	Hub      *WSHub
	UserID   string
	LastSeen time.Time
}

// WSHub represents the WebSocket connection hub
type WSHub struct {
	Clients    map[*WSClient]bool
	Broadcast  chan WSMessage
	Register   chan *WSClient
	Unregister chan *WSClient
}
